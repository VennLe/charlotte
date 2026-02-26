package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/spf13/viper"
)

// ConfigMigration 配置迁移工具
type ConfigMigration struct {
	sourcePath string
	targetPath string
}

// NewConfigMigration 创建配置迁移实例
func NewConfigMigration(sourcePath, targetPath string) *ConfigMigration {
	return &ConfigMigration{
		sourcePath: sourcePath,
		targetPath: targetPath,
	}
}

// MigrateFromEnvAndYaml 从.env和YAML配置迁移到新配置
func (cm *ConfigMigration) MigrateFromEnvAndYaml() error {
	fmt.Println("开始配置迁移...")

	// 1. 读取.env文件配置
	envConfig, err := cm.readEnvFile()
	if err != nil {
		return fmt.Errorf("读取.env文件失败: %v", err)
	}

	// 2. 读取YAML配置文件
	yamlConfig, err := cm.readYamlFiles()
	if err != nil {
		return fmt.Errorf("读取YAML文件失败: %v", err)
	}

	// 3. 合并配置（环境变量优先级高于YAML）
	mergedConfig := cm.mergeConfigs(envConfig, yamlConfig)

	// 4. 转换为新配置格式
	newConfig := cm.convertToNewFormat(mergedConfig)

	// 5. 保存新配置
	if err := cm.saveNewConfig(newConfig); err != nil {
		return fmt.Errorf("保存新配置失败: %v", err)
	}

	fmt.Println("配置迁移完成!")
	fmt.Printf("新配置文件: %s\n", cm.targetPath)
	fmt.Println("请检查新配置并根据需要调整安全设置。")

	return nil
}

// readEnvFile 读取.env文件
func (cm *ConfigMigration) readEnvFile() (map[string]interface{}, error) {
	envPath := filepath.Join(cm.sourcePath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return make(map[string]interface{}), nil
	}

	file, err := os.Open(envPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := make(map[string]interface{})
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// 转换为新配置格式的键名
		newKey := cm.convertEnvKeyToConfigKey(key)
		config[newKey] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	fmt.Printf("从.env文件读取了 %d 个配置项\n", len(config))
	return config, nil
}

// readYamlFiles 读取YAML配置文件
func (cm *ConfigMigration) readYamlFiles() (map[string]interface{}, error) {
	yamlPath := filepath.Join(cm.sourcePath, "configs")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return make(map[string]interface{}), nil
	}

	config := make(map[string]interface{})
	
	// 读取主要的config.yaml文件
	mainConfigPath := filepath.Join(yamlPath, "config.yaml")
	if _, err := os.Stat(mainConfigPath); err == nil {
		v := viper.New()
		v.SetConfigFile(mainConfigPath)
		
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
		
		// 将配置合并到map中
		for _, key := range v.AllKeys() {
			config[key] = v.Get(key)
		}
	}

	fmt.Printf("从YAML文件读取了 %d 个配置项\n", len(config))
	return config, nil
}

// convertEnvKeyToConfigKey 将.env键名转换为配置键名
func (cm *ConfigMigration) convertEnvKeyToConfigKey(envKey string) string {
	// 移除前缀并转换为小写
	key := strings.ToLower(envKey)
	
	// 常见映射关系
	mappings := map[string]string{
		"server_name":     "server.name",
		"server_port":     "server.port", 
		"server_mode":     "server.mode",
		"db_host":         "database.host",
		"db_port":         "database.port",
		"db_user":         "database.user",
		"db_password":     "database.password",
		"db_name":         "database.dbname",
		"redis_host":      "redis.host",
		"redis_port":      "redis.port",
		"redis_password":  "redis.password",
		"redis_db":        "redis.db",
		"kafka_broker_1":  "kafka.brokers.0",
		"jwt_secret":      "jwt.secret",
		"environment":     "environment",
	}
	
	if newKey, exists := mappings[key]; exists {
		return newKey
	}
	
	// 默认转换规则：下划线转点号
	return strings.ReplaceAll(key, "_", ".")
}

// mergeConfigs 合并配置（环境变量优先级高于YAML）
func (cm *ConfigMigration) mergeConfigs(envConfig, yamlConfig map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// 先添加YAML配置
	for key, value := range yamlConfig {
		merged[key] = value
	}
	
	// 环境变量覆盖YAML配置
	for key, value := range envConfig {
		merged[key] = value
	}
	
	return merged
}

// convertToNewFormat 转换为新配置格式
func (cm *ConfigMigration) convertToNewFormat(oldConfig map[string]interface{}) map[string]interface{} {
	newConfig := make(map[string]interface{})
	
	// 结构化的配置映射
	configStructure := map[string]string{
		"server.name":              "server.name",
		"server.port":              "server.port",
		"server.mode":              "server.mode",
		"server.base_url":          "server.base_url",
		"database.host":            "database.host",
		"database.port":            "database.port",
		"database.user":            "database.user",
		"database.password":        "database.password",
		"database.dbname":         "database.dbname",
		"database.max_open_conns":  "database.max_open_conns",
		"database.max_idle_conns": "database.max_idle_conns",
		"redis.host":               "redis.host",
		"redis.port":               "redis.port",
		"redis.password":           "redis.password",
		"redis.db":                 "redis.db",
		"redis.pool_size":          "redis.pool_size",
		"kafka.brokers.0":          "kafka.brokers.0",
		"kafka.topic":              "kafka.topic",
		"kafka.group_id":           "kafka.group_id",
		"jwt.secret":               "jwt.secret",
		"jwt.expire":               "jwt.expire",
		"jwt.issuer":               "jwt.issuer",
		"jwt.audience":             "jwt.audience",
		"log.level":                "log.level",
		"log.encoding":              "log.encoding",
		"security.cors_enabled":    "security.cors_enabled",
		"performance.request_timeout": "performance.request_timeout",
		"health.enabled":           "health.enabled",
		"file.upload_path":         "file.upload_path",
		"import_export.max_import_rows": "import_export.max_import_rows",
		"environment":              "environment",
	}
	
	for oldKey, newKey := range configStructure {
		if value, exists := oldConfig[oldKey]; exists {
			cm.setNestedValue(newConfig, newKey, value)
		}
	}
	
	return newConfig
}

// setNestedValue 设置嵌套配置值
func (cm *ConfigMigration) setNestedValue(config map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := config
	
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			current = current[part].(map[string]interface{})
		}
	}
}

// saveNewConfig 保存新配置
func (cm *ConfigMigration) saveNewConfig(config map[string]interface{}) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(cm.targetPath), 0755); err != nil {
		return err
	}
	
	// 转换为YAML格式
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	
	// 写入文件
	file, err := os.Create(cm.targetPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// 添加文件头注释
	header := `# Charlotte API 服务安全配置
# 此文件由配置迁移工具自动生成
# 支持环境变量覆盖和配置加密

`
	
	if _, err := file.WriteString(header); err != nil {
		return err
	}
	
	if _, err := file.Write(yamlData); err != nil {
		return err
	}
	
	return nil
}

// GenerateMigrationReport 生成迁移报告
func (cm *ConfigMigration) GenerateMigrationReport() string {
	report := "配置迁移报告\n"
	report += "=============\n\n"
	
	report += "迁移说明:\n"
	report += "- 从.env和多个YAML配置文件迁移到统一的安全配置系统\n"
	report += "- 支持环境变量覆盖配置值\n"
	report += "- 支持敏感配置加密存储\n"
	report += "- 简化配置结构，减少冗余\n\n"
	
	report += "迁移步骤:\n"
	report += "1. 运行迁移工具: go run cmd/migrate_config.go\n"
	report += "2. 检查生成的新配置文件\n"
	report += "3. 设置加密密钥（可选）\n"
	report += "4. 更新应用代码使用新配置系统\n\n"
	
	report += "安全建议:\n"
	report += "- 设置 CHARLOTTE_ENCRYPTION_KEY 环境变量启用配置加密\n"
	report += "- 使用密钥管理系统存储加密密钥\n"
	report += "- 定期轮换加密密钥\n"
	
	return report
}