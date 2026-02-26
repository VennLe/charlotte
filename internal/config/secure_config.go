package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/pkg/logger"
)

// SecureConfigManager 安全配置管理器
type SecureConfigManager struct {
	viper      *viper.Viper
	encryption *ConfigEncryption
	configPath string
}

// ConfigEncryption 配置加密组件
type ConfigEncryption struct {
	encryptionKey []byte
	enabled       bool
}

// NewSecureConfigManager 创建安全配置管理器
func NewSecureConfigManager() *SecureConfigManager {
	return &SecureConfigManager{
		viper: viper.New(),
		encryption: &ConfigEncryption{
			enabled: false,
		},
	}
}

// LoadSecureConfig 加载安全配置
func (scm *SecureConfigManager) LoadSecureConfig(configPath string) error {
	scm.configPath = configPath

	// 1. 设置环境变量优先级
	scm.setupEnvironment()

	// 2. 加载配置文件
	if err := scm.loadConfigFile(configPath); err != nil {
		return fmt.Errorf("配置文件加载失败: %v", err)
	}

	// 3. 解密敏感配置
	if err := scm.decryptSensitiveConfig(); err != nil {
		logger.Warn("配置解密失败，使用明文配置", zap.Error(err))
	}

	// 4. 验证配置完整性
	if err := scm.validateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	// 5. 设置配置热更新
	scm.setupConfigWatch()

	if logger.GetLogger() != nil {
		logger.Info("安全配置加载成功")
	}
	return nil
}

// setupEnvironment 设置环境变量
func (scm *SecureConfigManager) setupEnvironment() {
	scm.viper.AutomaticEnv()
	scm.viper.SetEnvPrefix("CHARLOTTE")
	scm.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// 设置环境变量优先级高于配置文件
	scm.viper.SetTypeByDefaultValue(true)
}

// loadConfigFile 加载配置文件
func (scm *SecureConfigManager) loadConfigFile(configPath string) error {
	if configPath != "" {
		scm.viper.SetConfigFile(configPath)
	} else {
		// 搜索配置文件路径
		scm.viper.AddConfigPath(".")
		scm.viper.AddConfigPath("./configs")
		scm.viper.AddConfigPath("/etc/charlotte")
		scm.viper.AddConfigPath("$HOME/.charlotte")
		scm.viper.SetConfigName("config")
		scm.viper.SetConfigType("yaml")
	}

	if err := scm.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Warn("配置文件未找到，使用环境变量和默认配置")
			return nil
		}
		return fmt.Errorf("配置文件读取失败: %v", err)
	}

	if logger.GetLogger() != nil {
		logger.Info("配置文件加载成功", zap.String("file", scm.viper.ConfigFileUsed()))
	}
	return nil
}

// decryptSensitiveConfig 解密敏感配置
func (scm *SecureConfigManager) decryptSensitiveConfig() error {
	// 检查是否启用加密
	encryptionKey := scm.viper.GetString("security.encryption_key")
	if encryptionKey == "" {
		return nil // 未启用加密
	}

	scm.encryption.enabled = true
	scm.encryption.encryptionKey = []byte(encryptionKey)

	// 需要解密的配置项
	sensitiveFields := []string{
		"database.password",
		"redis.password",
		"jwt.secret",
		"security.api_keys",
		"security.certificates",
	}

	for _, field := range sensitiveFields {
		if encryptedValue := scm.viper.GetString(field); encryptedValue != "" {
			decrypted, err := scm.decryptValue(encryptedValue)
			if err != nil {
				return fmt.Errorf("字段 %s 解密失败: %v", field, err)
			}
			scm.viper.Set(field, decrypted)
		}
	}

	return nil
}

// encryptValue 加密值
func (scm *SecureConfigManager) encryptValue(value string) (string, error) {
	if !scm.encryption.enabled {
		return value, nil
	}

	block, err := aes.NewCipher(scm.encryption.encryptionKey)
	if err != nil {
		return "", err
	}

	// 使用GCM模式加密
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptValue 解密值
func (scm *SecureConfigManager) decryptValue(encryptedValue string) (string, error) {
	if !scm.encryption.enabled {
		return encryptedValue, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(scm.encryption.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("密文长度无效")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// validateConfig 验证配置完整性
func (scm *SecureConfigManager) validateConfig() error {
	// 检查必需配置
	requiredConfigs := map[string]string{
		"server.name":     "服务器名称",
		"server.port":     "服务器端口",
		"database.host":   "数据库主机",
		"database.user":   "数据库用户",
		"database.dbname": "数据库名称",
		"jwt.secret":      "JWT密钥",
	}

	for key, description := range requiredConfigs {
		if scm.viper.GetString(key) == "" {
			return fmt.Errorf("必需配置缺失: %s (%s)", description, key)
		}
	}

	// 验证JWT密钥长度
	jwtSecret := scm.viper.GetString("jwt.secret")
	if len(jwtSecret) < 32 {
		logger.Warn("JWT密钥长度建议至少32位，当前密钥安全性较低")
	}

	return nil
}

// setupConfigWatch 设置配置热更新监听
func (scm *SecureConfigManager) setupConfigWatch() {
	scm.viper.WatchConfig()
	scm.viper.OnConfigChange(func(e fsnotify.Event) {
		logger.Info("配置文件已更新，重新加载配置", zap.String("file", e.Name))

		// 重新加载配置
		if err := scm.LoadSecureConfig(scm.configPath); err != nil {
			logger.Error("配置热更新失败", zap.Error(err))
		} else {
			logger.Info("配置热更新成功")
		}
	})
}

// GetConfig 获取配置值
func (scm *SecureConfigManager) GetConfig(key string) interface{} {
	return scm.viper.Get(key)
}

// GetString 获取字符串配置值
func (scm *SecureConfigManager) GetString(key string) string {
	return scm.viper.GetString(key)
}

// GetInt 获取整数配置值
func (scm *SecureConfigManager) GetInt(key string) int {
	return scm.viper.GetInt(key)
}

// GetBool 获取布尔配置值
func (scm *SecureConfigManager) GetBool(key string) bool {
	return scm.viper.GetBool(key)
}

// GetStringSlice 获取字符串切片配置值
func (scm *SecureConfigManager) GetStringSlice(key string) []string {
	return scm.viper.GetStringSlice(key)
}

// SaveSecureConfig 保存加密配置
func (scm *SecureConfigManager) SaveSecureConfig(configPath string, configData map[string]interface{}) error {
	// 加密敏感数据
	encryptedConfig := make(map[string]interface{})
	for key, value := range configData {
		if scm.isSensitiveField(key) {
			if strValue, ok := value.(string); ok {
				encrypted, err := scm.encryptValue(strValue)
				if err != nil {
					return fmt.Errorf("加密字段 %s 失败: %v", key, err)
				}
				encryptedConfig[key] = encrypted
			} else {
				encryptedConfig[key] = value
			}
		} else {
			encryptedConfig[key] = value
		}
	}

	// 保存到文件
	return scm.saveConfigToFile(configPath, encryptedConfig)
}

// isSensitiveField 判断是否为敏感字段
func (scm *SecureConfigManager) isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"password",
		"secret",
		"key",
		"token",
		"certificate",
		"private",
	}

	fieldLower := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

// saveConfigToFile 保存配置到文件
func (scm *SecureConfigManager) saveConfigToFile(configPath string, configData map[string]interface{}) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	tempViper := viper.New()
	for key, value := range configData {
		tempViper.Set(key, value)
	}

	return tempViper.WriteConfigAs(configPath)
}

// GetConfigSummary 获取配置摘要
func (scm *SecureConfigManager) GetConfigSummary() string {
	return fmt.Sprintf(`
配置摘要:
  服务器: %s:%s (%s)
  数据库: %s@%s:%s/%s
  Redis:  %s:%s
  JWT:    %d小时过期
  配置源: %s
`,
		scm.GetString("server.name"),
		scm.GetString("server.port"),
		scm.GetString("server.mode"),
		scm.GetString("database.user"),
		scm.GetString("database.host"),
		scm.GetString("database.port"),
		scm.GetString("database.dbname"),
		scm.GetString("redis.host"),
		scm.GetString("redis.port"),
		scm.GetInt("jwt.expire"),
		scm.viper.ConfigFileUsed(),
	)
}
