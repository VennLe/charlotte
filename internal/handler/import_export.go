package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/service"
	"github.com/VennLe/charlotte/pkg/logger"
	"github.com/VennLe/charlotte/pkg/utils"
)

// ImportExportHandler 导入导出处理器
type ImportExportHandler struct {
	importExportService *service.ImportExportService
	fileService         *service.FileService
}

// NewImportExportHandler 创建导入导出处理器
func NewImportExportHandler(
	importExportService *service.ImportExportService,
	fileService *service.FileService,
) *ImportExportHandler {
	return &ImportExportHandler{
		importExportService: importExportService,
		fileService:         fileService,
	}
}

// ImportData 导入数据
func (h *ImportExportHandler) ImportData(c *gin.Context) {
	var req service.ImportRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	// 根据数据类型选择处理器
	processor, err := h.getDataProcessor(req.DataType)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 执行导入
	resp, err := h.importExportService.ImportData(c.Request.Context(), &req, processor)
	if err != nil {
		logger.Error("数据导入失败",
			zap.String("data_type", req.DataType),
			zap.String("file_type", req.FileType),
			zap.Error(err),
		)
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !resp.Success {
		utils.Error(c, http.StatusBadRequest, resp.Message)
		return
	}

	logger.Info("数据导入完成",
		zap.String("data_type", req.DataType),
		zap.String("file_type", req.FileType),
		zap.Int("total_rows", resp.TotalRows),
		zap.Int("success_rows", resp.SuccessRows),
		zap.Int("failed_rows", resp.FailedRows),
		zap.Uint("user_id", userID.(uint)),
	)

	utils.Success(c, resp)
}

// ExportData 导出数据
func (h *ImportExportHandler) ExportData(c *gin.Context) {
	var req service.ExportRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	// 根据数据类型选择处理器
	processor, err := h.getDataProcessor(req.DataType)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 执行导出
	resp, err := h.importExportService.ExportData(c.Request.Context(), &req, processor)
	if err != nil {
		logger.Error("数据导出失败",
			zap.String("data_type", req.DataType),
			zap.String("file_type", req.FileType),
			zap.Error(err),
		)
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("数据导出成功",
		zap.String("data_type", req.DataType),
		zap.String("file_type", req.FileType),
		zap.String("file_name", resp.FileName),
		zap.Int("file_size", resp.FileSize),
		zap.Uint("user_id", userID.(uint)),
	)

	// 设置响应头
	c.Header("Content-Disposition", "attachment; filename="+resp.FileName)
	c.Header("Content-Type", h.getContentType(resp.FileType))
	c.Header("Content-Length", strconv.Itoa(resp.FileSize))

	// 发送文件数据
	c.Data(http.StatusOK, h.getContentType(resp.FileType), resp.Data)
}

// UploadFile 上传文件
func (h *ImportExportHandler) UploadFile(c *gin.Context) {
	var req service.UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	userName, exists := c.Get("username")
	if !exists {
		userName = "unknown"
	}

	// 执行文件上传
	resp, err := h.fileService.UploadFile(c.Request.Context(), &req, userID.(uint), userName.(string))
	if err != nil {
		logger.Error("文件上传失败",
			zap.String("filename", req.File.Filename),
			zap.Int64("size", req.File.Size),
			zap.Error(err),
		)
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("文件上传成功",
		zap.String("file_id", resp.FileInfo.ID),
		zap.String("filename", resp.FileInfo.OriginalName),
		zap.Int64("size", resp.FileInfo.Size),
		zap.Uint("user_id", userID.(uint)),
	)

	utils.Success(c, resp)
}

// DownloadFile 下载文件
func (h *ImportExportHandler) DownloadFile(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		utils.Error(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	// 执行文件下载
	fileInfo, file, err := h.fileService.DownloadFile(c.Request.Context(), fileID)
	if err != nil {
		logger.Error("文件下载失败",
			zap.String("file_id", fileID),
			zap.Error(err),
		)
		utils.Error(c, http.StatusNotFound, "文件不存在或已删除")
		return
	}
	defer file.Close()

	logger.Info("文件下载成功",
		zap.String("file_id", fileID),
		zap.String("filename", fileInfo.Name),
		zap.Int64("size", fileInfo.Size),
	)

	// 设置响应头
	c.Header("Content-Disposition", "attachment; filename="+fileInfo.OriginalName)
	c.Header("Content-Type", fileInfo.MimeType)
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size, 10))

	// 发送文件
	c.File(fileInfo.Path)
}

// ListFiles 列出文件
func (h *ImportExportHandler) ListFiles(c *gin.Context) {
	var req service.ListFilesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 || req.Size > 100 {
		req.Size = 20
	}

	// 执行文件列表查询
	resp, err := h.fileService.ListFiles(c.Request.Context(), &req)
	if err != nil {
		logger.Error("获取文件列表失败",
			zap.String("category", req.Category),
			zap.Error(err),
		)
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, resp)
}

// DeleteFile 删除文件
func (h *ImportExportHandler) DeleteFile(c *gin.Context) {
	var req service.DeleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT中获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "未登录")
		return
	}

	// 执行文件删除
	err := h.fileService.DeleteFile(c.Request.Context(), req.FileID)
	if err != nil {
		logger.Error("文件删除失败",
			zap.String("file_id", req.FileID),
			zap.Error(err),
		)
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("文件删除成功",
		zap.String("file_id", req.FileID),
		zap.Uint("user_id", userID.(uint)),
	)

	utils.Success(c, gin.H{"message": "文件删除成功"})
}

// GetSupportedDataTypes 获取支持的数据类型
func (h *ImportExportHandler) GetSupportedDataTypes(c *gin.Context) {
	dataTypes := h.importExportService.GetSupportedDataTypes()
	fileTypes := h.importExportService.GetSupportedFileTypes()

	utils.Success(c, gin.H{
		"data_types": dataTypes,
		"file_types": fileTypes,
	})
}

// getDataProcessor 根据数据类型获取处理器
func (h *ImportExportHandler) getDataProcessor(dataType string) (service.DataProcessor, error) {
	switch dataType {
	case "user":
		return &service.UserDataProcessor{}, nil
	// 可以添加更多数据类型处理器
	// case "product":
	//     return &service.ProductDataProcessor{}, nil
	// case "order":
	//     return &service.OrderDataProcessor{}, nil
	default:
		return nil, fmt.Errorf("不支持的数据类型: %s", dataType)
	}
}

// getContentType 根据文件类型获取Content-Type
func (h *ImportExportHandler) getContentType(fileType string) string {
	switch fileType {
	case "csv":
		return "text/csv; charset=utf-8"
	case "excel":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "json":
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// GetFileInfo 获取文件信息
func (h *ImportExportHandler) GetFileInfo(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		utils.Error(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	// 这里应该从数据库查询文件信息
	// 简化实现：直接返回文件信息
	fileInfo := &service.FileInfo{
		ID:           fileID,
		Name:         "示例文件",
		OriginalName: "example.txt",
		Size:         1024,
		MimeType:     "text/plain",
		Extension:    ".txt",
		UploadTime:   time.Now(),
	}

	utils.Success(c, fileInfo)
}

// GetImportTemplate 获取导入模板
func (h *ImportExportHandler) GetImportTemplate(c *gin.Context) {
	dataType := c.Query("data_type")
	fileType := c.Query("file_type")

	if dataType == "" || fileType == "" {
		utils.Error(c, http.StatusBadRequest, "数据类型和文件类型不能为空")
		return
	}

	// 获取处理器
	processor, err := h.getDataProcessor(dataType)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 创建空数据用于生成模板
	emptyData := processor.CreateEmptySlice()

	// 配置导出参数
	exportConfig := &utils.ExportConfig{
		FileType: fileType,
		FileName: fmt.Sprintf("%s_template.%s", dataType, fileType),
		Headers:  processor.GetExportHeaders(),
		FieldMap: processor.GetExportFieldMap(),
	}

	// 生成模板文件
	fileData, err := utils.ExportData(emptyData, exportConfig)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "生成模板失败: "+err.Error())
		return
	}

	// 设置响应头
	c.Header("Content-Disposition", "attachment; filename="+exportConfig.FileName)
	c.Header("Content-Type", h.getContentType(fileType))
	c.Header("Content-Length", strconv.Itoa(len(fileData)))

	// 发送模板文件
	c.Data(http.StatusOK, h.getContentType(fileType), fileData)
}
