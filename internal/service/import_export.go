package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/VennLe/charlotte/pkg/utils"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/pkg/logger"
)

// ImportExportService 导入导出服务
type ImportExportService struct {
	fileService *FileService
}

// NewImportExportService 创建导入导出服务
func NewImportExportService(fileService *FileService) *ImportExportService {
	return &ImportExportService{
		fileService: fileService,
	}
}

// ImportRequest 导入请求
type ImportRequest struct {
	File       *multipart.FileHeader `form:"file" binding:"required"`
	FileType   string                `form:"file_type" binding:"required,oneof=csv excel json"`
	DataType   string                `form:"data_type" binding:"required"` // 数据类型标识
	HasHeader  bool                  `form:"has_header"`                   // 是否有表头
	StartRow   int                   `form:"start_row" default:"1"`        // 数据开始行
	SheetName  string                `form:"sheet_name"`                   // Excel工作表名
	DateFormat string                `form:"date_format"`                  // 日期格式
	TimeFormat string                `form:"time_format"`                  // 时间格式
}

// ExportRequest 导出请求
type ExportRequest struct {
	DataType   string      `form:"data_type" binding:"required"` // 数据类型标识
	FileType   string      `form:"file_type" binding:"required,oneof=csv excel json"`
	FileName   string      `form:"file_name"`   // 文件名
	Headers    []string    `form:"headers"`     // 表头
	FieldMap   string      `form:"field_map"`   // 字段映射JSON
	DateFormat string      `form:"date_format"` // 日期格式
	TimeFormat string      `form:"time_format"` // 时间格式
	Data       interface{} `json:"data"`        // 要导出的数据
}

// ImportResponse 导入响应
type ImportResponse struct {
	Success     bool                       `json:"success"`
	Message     string                     `json:"message"`
	TotalRows   int                        `json:"total_rows"`
	SuccessRows int                        `json:"success_rows"`
	FailedRows  int                        `json:"failed_rows"`
	Errors      []*utils.ImportExportError `json:"errors,omitempty"`
	Data        interface{}                `json:"data,omitempty"`
}

// ExportResponse 导出响应
type ExportResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	FileName string `json:"file_name"`
	FileSize int    `json:"file_size"`
	FileType string `json:"file_type"`
	Data     []byte `json:"-"` // 文件数据
}

// DataProcessor 数据处理接口
type DataProcessor interface {
	// GetDataType 获取数据类型标识
	GetDataType() string
	// CreateEmptySlice 创建空的数据切片
	CreateEmptySlice() interface{}
	// ValidateData 验证数据
	ValidateData(data interface{}) error
	// ProcessData 处理导入的数据
	ProcessData(ctx context.Context, data interface{}) error
	// GetExportData 获取导出的数据
	GetExportData(ctx context.Context, params map[string]interface{}) (interface{}, error)
	// GetExportHeaders 获取导出表头
	GetExportHeaders() []string
	// GetExportFieldMap 获取导出字段映射
	GetExportFieldMap() map[string]string
}

// ImportData 通用数据导入
func (s *ImportExportService) ImportData(ctx context.Context, req *ImportRequest, processor DataProcessor) (*ImportResponse, error) {
	// 验证数据类型
	if processor.GetDataType() != req.DataType {
		return nil, fmt.Errorf("数据类型不匹配: %s != %s", processor.GetDataType(), req.DataType)
	}

	// 创建空的数据切片
	dataSlice := processor.CreateEmptySlice()

	// 配置导入参数
	importConfig := &utils.ImportConfig{
		FileType:   req.FileType,
		HasHeader:  req.HasHeader,
		StartRow:   req.StartRow,
		SheetName:  req.SheetName,
		DateFormat: req.DateFormat,
		TimeFormat: req.TimeFormat,
	}

	// 执行导入
	result, err := utils.ImportData(dataSlice, req.File, importConfig)
	if err != nil {
		return nil, fmt.Errorf("导入失败: %v", err)
	}

	// 验证数据
	if err := processor.ValidateData(dataSlice); err != nil {
		return &ImportResponse{
			Success:     false,
			Message:     "数据验证失败: " + err.Error(),
			TotalRows:   result.TotalRows,
			SuccessRows: 0,
			FailedRows:  result.TotalRows,
			Errors:      result.Errors,
		}, nil
	}

	// 处理数据
	if err := processor.ProcessData(ctx, dataSlice); err != nil {
		return &ImportResponse{
			Success:     false,
			Message:     "数据处理失败: " + err.Error(),
			TotalRows:   result.TotalRows,
			SuccessRows: 0,
			FailedRows:  result.TotalRows,
			Errors:      result.Errors,
		}, nil
	}

	logger.Info("数据导入成功",
		zap.String("data_type", req.DataType),
		zap.String("file_type", req.FileType),
		zap.Int("total_rows", result.TotalRows),
		zap.Int("success_rows", result.SuccessRows),
		zap.Int("failed_rows", result.FailedRows),
	)

	return &ImportResponse{
		Success:     true,
		Message:     "数据导入成功",
		TotalRows:   result.TotalRows,
		SuccessRows: result.SuccessRows,
		FailedRows:  result.FailedRows,
		Errors:      result.Errors,
		Data:        dataSlice,
	}, nil
}

// ExportData 通用数据导出
func (s *ImportExportService) ExportData(ctx context.Context, req *ExportRequest, processor DataProcessor) (*ExportResponse, error) {
	// 验证数据类型
	if processor.GetDataType() != req.DataType {
		return nil, fmt.Errorf("数据类型不匹配: %s != %s", processor.GetDataType(), req.DataType)
	}

	// 获取导出数据
	params := make(map[string]interface{})
	if req.Data != nil {
		// 如果直接提供了数据，则使用提供的数据
		params["data"] = req.Data
	} else {
		// 否则从处理器获取数据
		data, err := processor.GetExportData(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("获取导出数据失败: %v", err)
		}
		req.Data = data
	}

	// 配置导出参数
	exportConfig := &utils.ExportConfig{
		FileType:   req.FileType,
		FileName:   req.FileName,
		DateFormat: req.DateFormat,
		TimeFormat: req.TimeFormat,
	}

	// 设置表头
	if len(req.Headers) > 0 {
		exportConfig.Headers = req.Headers
	} else {
		exportConfig.Headers = processor.GetExportHeaders()
	}

	// 设置字段映射
	if req.FieldMap != "" {
		var fieldMap map[string]string
		if err := json.Unmarshal([]byte(req.FieldMap), &fieldMap); err != nil {
			return nil, fmt.Errorf("解析字段映射失败: %v", err)
		}
		exportConfig.FieldMap = fieldMap
	} else {
		exportConfig.FieldMap = processor.GetExportFieldMap()
	}

	// 生成文件名
	if exportConfig.FileName == "" {
		exportConfig.FileName = s.generateFileName(req.DataType, req.FileType)
	}

	// 执行导出
	fileData, err := utils.ExportData(req.Data, exportConfig)
	if err != nil {
		return nil, fmt.Errorf("导出失败: %v", err)
	}

	logger.Info("数据导出成功",
		zap.String("data_type", req.DataType),
		zap.String("file_type", req.FileType),
		zap.String("file_name", exportConfig.FileName),
		zap.Int("file_size", len(fileData)),
	)

	return &ExportResponse{
		Success:  true,
		Message:  "数据导出成功",
		FileName: exportConfig.FileName,
		FileSize: len(fileData),
		FileType: req.FileType,
		Data:     fileData,
	}, nil
}

// generateFileName 生成文件名
func (s *ImportExportService) generateFileName(dataType, fileType string) string {
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s_%s.%s", dataType, timestamp, strings.ToLower(fileType))
}

// UserDataProcessor 用户数据处理器示例
type UserDataProcessor struct{}

func (p *UserDataProcessor) GetDataType() string {
	return "user"
}

func (p *UserDataProcessor) CreateEmptySlice() interface{} {
	return &[]UserInfo{}
}

func (p *UserDataProcessor) ValidateData(data interface{}) error {
	users, ok := data.(*[]UserInfo)
	if !ok {
		return fmt.Errorf("数据类型错误，期望*[]UserInfo")
	}

	for i, user := range *users {
		if user.Username == "" {
			return fmt.Errorf("第%d行: 用户名不能为空", i+1)
		}
		if user.Email == "" {
			return fmt.Errorf("第%d行: 邮箱不能为空", i+1)
		}
		// 更多验证规则...
	}

	return nil
}

func (p *UserDataProcessor) ProcessData(ctx context.Context, data interface{}) error {
	_, ok := data.(*[]UserInfo)
	if !ok {
		return fmt.Errorf("数据类型错误，期望*[]UserInfo")
	}

	// 这里可以添加业务逻辑，如保存到数据库等
	// 示例：批量创建用户
	// userService := NewUserService(db)
	// users := data.(*[]UserInfo)
	// for _, userInfo := range *users {
	//     req := &RegisterRequest{
	//         Username: userInfo.Username,
	//         Email:    userInfo.Email,
	//         // ... 其他字段
	//     }
	//     if _, err := userService.Register(ctx, req); err != nil {
	//         return err
	//     }
	// }

	return nil
}

func (p *UserDataProcessor) GetExportData(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 这里可以从数据库获取用户数据
	// 示例：
	// userService := NewUserService(db)
	// page := 1
	// size := 1000
	// if p, ok := params["page"].(int); ok {
	//     page = p
	// }
	// if s, ok := params["size"].(int); ok {
	//     size = s
	// }
	//
	// users, _, err := userService.GetUserList(ctx, page, size, "")
	// return users, err

	// 返回空数据示例
	return []UserInfo{}, nil
}

func (p *UserDataProcessor) GetExportHeaders() []string {
	return []string{
		"ID",
		"用户名",
		"邮箱",
		"昵称",
		"手机号",
		"角色",
		"状态",
		"最后登录时间",
		"创建时间",
	}
}

func (p *UserDataProcessor) GetExportFieldMap() map[string]string {
	return map[string]string{
		"ID":        "id",
		"Username":  "username",
		"Email":     "email",
		"Nickname":  "nickname",
		"Phone":     "phone",
		"Role":      "role",
		"Status":    "status",
		"LastLogin": "last_login",
		"CreatedAt": "created_at",
	}
}

// RegisterDataProcessor 注册数据处理器
func (s *ImportExportService) RegisterDataProcessor(dataType string, processor DataProcessor) error {
	// 在实际应用中，这里应该维护一个处理器映射表
	// 这里简化实现
	return nil
}

// GetSupportedDataTypes 获取支持的数据类型
func (s *ImportExportService) GetSupportedDataTypes() []string {
	return []string{
		"user",     // 用户数据
		"product",  // 产品数据
		"order",    // 订单数据
		"customer", // 客户数据
		// 可以扩展更多数据类型
	}
}

// GetSupportedFileTypes 获取支持的文件类型
func (s *ImportExportService) GetSupportedFileTypes() []string {
	return []string{
		"csv",
		"excel",
		"json",
	}
}
