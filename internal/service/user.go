package service

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/internal/dao"
	"github.com/VennLe/charlotte/internal/model"
	"github.com/VennLe/charlotte/pkg/kafka"
	"github.com/VennLe/charlotte/pkg/logger"
)

// UserService 用户服务
type UserService struct {
	dao      *dao.UserDAO
	producer kafka.Producer
}

// NewUserService 创建服务实例
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		dao:      dao.NewUserDAO(db),
		producer: kafka.GetProducer(),
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=32"`
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expires_at"`
	User      *model.User `json:"user"`
}

// UserInfo 用户信息 (脱敏)
type UserInfo struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Phone     string    `json:"phone"`
	Status    int       `json:"status"`
	Role      string    `json:"role"`
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
		Phone:    req.Phone,
	}

	if err := s.dao.Create(ctx, user); err != nil {
		return nil, err
	}

	// 发送 Kafka 事件
	go s.publishUserEvent("user_created", user)

	return user, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 先尝试用户名登录，再尝试邮箱登录
	user, err := s.dao.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, dao.ErrUserNotFound) {
			// 尝试邮箱登录
			user, err = s.dao.GetByEmail(ctx, req.Username)
			if err != nil {
				return nil, errors.New("用户名或密码错误")
			}
		} else {
			return nil, err
		}
	}

	// 检查状态
	if user.Status != 1 {
		return nil, errors.New("账号已被禁用")
	}

	// 验证密码
	if !s.dao.CheckPassword(user.Password, req.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	// 更新最后登录时间
	go s.dao.UpdateLastLogin(ctx, user.ID)

	// 生成 JWT
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	// 脱敏处理
	user.Password = ""

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// GetUserByID 获取用户信息
func (s *UserService) GetUserByID(ctx context.Context, id uint) (*UserInfo, error) {
	user, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toUserInfo(user), nil
}

// GetUserList 获取用户列表
func (s *UserService) GetUserList(ctx context.Context, page, size int, keyword string) ([]*UserInfo, int64, error) {
	users, total, err := s.dao.List(ctx, page, size, keyword)
	if err != nil {
		return nil, 0, err
	}

	var list []*UserInfo
	for _, user := range users {
		list = append(list, s.toUserInfo(user))
	}

	return list, total, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	// 检查用户是否存在
	_, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.dao.Update(ctx, id, updates); err != nil {
		return err
	}

	// 发送更新事件
	go func() {
		user, _ := s.dao.GetByID(context.Background(), id)
		if user != nil {
			s.publishUserEvent("user_updated", user)
		}
	}()

	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	user, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.dao.Delete(ctx, id); err != nil {
		return err
	}

	// 发送删除事件
	go s.publishUserEvent("user_deleted", user)

	return nil
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(ctx context.Context, id uint, oldPassword, newPassword string) error {
	user, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 验证旧密码
	if !s.dao.CheckPassword(user.Password, oldPassword) {
		return errors.New("原密码错误")
	}

	return s.dao.UpdatePassword(ctx, id, newPassword)
}

// generateToken 生成 JWT Token
func (s *UserService) generateToken(user *model.User) (string, int64, error) {
	expireHours := config.Global.JWT.Expire
	if expireHours == 0 {
		expireHours = 24
	}

	expiresAt := time.Now().Add(time.Hour * time.Duration(expireHours)).Unix()

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      expiresAt,
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.Global.JWT.Secret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt, nil
}

// toUserInfo 将User转换为脱敏的UserInfo
func (s *UserService) toUserInfo(user *model.User) *UserInfo {
	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Phone:     user.Phone,
		Status:    user.Status,
		Role:      user.Role,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
	}
}

// publishUserEvent 发送用户事件到 Kafka
func (s *UserService) publishUserEvent(eventType string, user *model.User) {
	event := model.UserEvent{
		EventType: eventType,
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Timestamp: time.Now(),
	}

	// 序列化完整数据
	data, _ := json.Marshal(user)
	event.Data = string(data)

	eventJSON, _ := json.Marshal(event)

	if err := s.producer.SendMessage("user-events", string(eventJSON)); err != nil {
		logger.Error("发送用户事件失败",
			zap.String("event_type", eventType),
			zap.Uint("user_id", user.ID),
			zap.Error(err))
	} else {
		logger.Info("用户事件发送成功",
			zap.String("event_type", eventType),
			zap.Uint("user_id", user.ID))
	}
}
