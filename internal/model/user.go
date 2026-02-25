package model

import (
	"gorm.io/gorm"
	"time"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Username  string    `gorm:"size:50;not null;uniqueIndex" json:"username" binding:"required,min=3,max=50"`
	Email     string    `gorm:"size:100;not null;uniqueIndex" json:"email" binding:"required,email"`
	Password  string    `gorm:"size:255;not null" json:"-"` // 密码不序列化
	Nickname  string    `gorm:"size:50" json:"nickname"`
	Avatar    string    `gorm:"size:255" json:"avatar"`
	Phone     string    `gorm:"size:20" json:"phone"`
	Status    int       `gorm:"default:1;comment:1正常 2禁用" json:"status"`
	Role      string    `gorm:"size:20;default:user" json:"role"` // admin/user
	LastLogin time.Time `json:"last_login"`
}

// UserEvent 用户事件 (用于 Kafka)
type UserEvent struct {
	EventType string    `json:"event_type"` // user_created, user_updated, user_deleted
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"` // JSON 格式的完整数据
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = "user"
	}
	if u.Status == 0 {
		u.Status = 1
	}
	return nil
}
