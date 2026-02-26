package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

// FileService 文件服务
type FileService struct {
	basePath string
}

// NewFileService 创建文件服务
func NewFileService() *FileService {
	basePath := config.Global.File.UploadPath
	if basePath == "" {
		basePath = "resources"
	}

	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		logger.Error("创建上传目录失败", zap.String("path", basePath), zap.Error(err))
	}

	return &FileService{
		basePath: basePath,
	}
}

// FileInfo 文件信息
type FileInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OriginalName string    `json:"original_name"`
	Size        int64     `json:"size"`
	MimeType    string    `json:"mime_type"`
	Extension   string    `json:"extension"`
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	MD5         string    `json:"md5"`
	UploadTime  time.Time `json:"upload_time"`
	UploaderID  uint      `json:"uploader_id,omitempty"`
	UploaderName string    `json:"uploader_name,omitempty"`
}

// UploadRequest 上传请求
type UploadRequest struct {
	File        *multipart.FileHeader `form:"file" binding:"required"`
	Category    string                `form:"category"` // 文件分类
	Description string                `form:"description"`
	IsPublic    bool                  `form:"is_public"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	FileInfo *FileInfo `json:"file_info"`
	Message  string    `json:"message"`
}

// DownloadRequest 下载请求
type DownloadRequest struct {
	FileID string `form:"file_id" binding:"required"`
}

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	Category string `form:"category"`
	Page     int    `form:"page" default:"1"`
	Size     int    `form:"size" default:"20"`
	Keyword  string `form:"keyword"`
}

// ListFilesResponse 文件列表响应
type ListFilesResponse struct {
	Files []*FileInfo `json:"files"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// DeleteFileRequest 删除文件请求
type DeleteFileRequest struct {
	FileID string `json:"file_id" binding:"required"`
}

// UploadFile 上传文件
func (s *FileService) UploadFile(ctx context.Context, req *UploadRequest, uploaderID uint, uploaderName string) (*UploadResponse, error) {
	// 验证文件大小
	maxSize := config.Global.File.MaxUploadSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024 // 默认10MB
	}

	if req.File.Size > maxSize {
		return nil, fmt.Errorf("文件大小超过限制: %d > %d", req.File.Size, maxSize)
	}

	// 验证文件类型
	if err := s.validateFileType(req.File); err != nil {
		return nil, err
	}

	// 打开文件
	srcFile, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer srcFile.Close()

	// 计算MD5
	hash := md5.New()
	if _, err := io.Copy(hash, srcFile); err != nil {
		return nil, fmt.Errorf("计算文件MD5失败: %v", err)
	}

	// 重置文件指针
	srcFile.Seek(0, 0)

	md5sum := hex.EncodeToString(hash.Sum(nil))

	// 生成文件ID和路径
	fileID := s.generateFileID(req.File.Filename, md5sum)
	filePath := s.generateFilePath(fileID, req.File.Filename, req.Category)

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}

	// 保存文件
	dstFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return nil, fmt.Errorf("保存文件失败: %v", err)
	}

	// 构建文件信息
	fileInfo := &FileInfo{
		ID:           fileID,
		Name:         filepath.Base(filePath),
		OriginalName: req.File.Filename,
		Size:         req.File.Size,
		MimeType:     req.File.Header.Get("Content-Type"),
		Extension:    strings.ToLower(filepath.Ext(req.File.Filename)),
		Path:         filePath,
		URL:          s.generateFileURL(fileID),
		MD5:          md5sum,
		UploadTime:   time.Now(),
		UploaderID:   uploaderID,
		UploaderName: uploaderName,
	}

	logger.Info("文件上传成功",
		zap.String("file_id", fileID),
		zap.String("filename", req.File.Filename),
		zap.Int64("size", req.File.Size),
		zap.String("md5", md5sum),
		zap.Uint("uploader_id", uploaderID),
	)

	return &UploadResponse{
		FileInfo: fileInfo,
		Message:  "文件上传成功",
	}, nil
}

// DownloadFile 下载文件
func (s *FileService) DownloadFile(ctx context.Context, fileID string) (*FileInfo, *os.File, error) {
	// 查找文件
	filePath, err := s.findFilePath(fileID)
	if err != nil {
		return nil, nil, err
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("打开文件失败: %v", err)
	}

	// 获取文件信息
	fileStat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	fileInfo := &FileInfo{
		ID:          fileID,
		Name:        filepath.Base(filePath),
		Size:        fileStat.Size(),
		Path:        filePath,
		UploadTime:  fileStat.ModTime(),
	}

	return fileInfo, file, nil
}

// ListFiles 列出文件
func (s *FileService) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
	// 在实际应用中，这里应该查询数据库
	// 这里简化实现，直接扫描目录

	var files []*FileInfo
	baseDir := s.basePath

	if req.Category != "" {
		baseDir = filepath.Join(s.basePath, req.Category)
	}

	// 扫描目录
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// 简化实现，实际应该从数据库查询
			fileInfo := &FileInfo{
				ID:         s.generateFileIDFromPath(path),
				Name:       info.Name(),
				Size:       info.Size(),
				Path:       path,
				UploadTime: info.ModTime(),
			}

			files = append(files, fileInfo)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("扫描文件目录失败: %v", err)
	}

	// 分页处理
	total := len(files)
	start := (req.Page - 1) * req.Size
	end := start + req.Size

	if start >= total {
		return &ListFilesResponse{
			Files: []*FileInfo{},
			Total: int64(total),
			Page:  req.Page,
			Size:  req.Size,
		}, nil
	}

	if end > total {
		end = total
	}

	return &ListFilesResponse{
		Files: files[start:end],
		Total: int64(total),
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// DeleteFile 删除文件
func (s *FileService) DeleteFile(ctx context.Context, fileID string) error {
	filePath, err := s.findFilePath(fileID)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}

	logger.Info("文件删除成功",
		zap.String("file_id", fileID),
		zap.String("file_path", filePath),
	)

	return nil
}

// validateFileType 验证文件类型
func (s *FileService) validateFileType(file *multipart.FileHeader) error {
	allowedTypes := config.Global.File.AllowedTypes
	if len(allowedTypes) == 0 {
		// 默认允许的文件类型
		allowedTypes = []string{
			"image/jpeg", "image/png", "image/gif", "image/webp",
			"application/pdf", "application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain", "text/csv", "application/json",
		}
	}

	contentType := file.Header.Get("Content-Type")
	for _, allowedType := range allowedTypes {
		if contentType == allowedType {
			return nil
		}
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".txt", ".csv", ".json",
	}

	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			return nil
		}
	}

	return fmt.Errorf("不支持的文件类型: %s", contentType)
}

// generateFileID 生成文件ID
func (s *FileService) generateFileID(filename, md5sum string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d", md5sum[:8], timestamp)
}

// generateFilePath 生成文件路径
func (s *FileService) generateFilePath(fileID, filename, category string) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	if category == "" {
		category = "general"
	}

	// 路径格式: resources/category/year/month/day/fileID_original.ext
	path := filepath.Join(
		s.basePath,
		category,
		year,
		month,
		day,
		fmt.Sprintf("%s%s", fileID, filepath.Ext(filename)),
	)

	return path
}

// generateFileURL 生成文件访问URL
func (s *FileService) generateFileURL(fileID string) string {
	baseURL := config.Global.Server.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/api/v1/files/download/%s", baseURL, fileID)
}

// findFilePath 查找文件路径
func (s *FileService) findFilePath(fileID string) (string, error) {
	// 在实际应用中，这里应该查询数据库获取文件路径
	// 这里简化实现，扫描目录查找文件

	var foundPath string
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			filename := info.Name()
			if strings.HasPrefix(filename, fileID) {
				foundPath = path
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if foundPath == "" {
		return "", fmt.Errorf("文件不存在: %s", fileID)
	}

	return foundPath, nil
}

// generateFileIDFromPath 从文件路径生成文件ID
func (s *FileService) generateFileIDFromPath(path string) string {
	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}