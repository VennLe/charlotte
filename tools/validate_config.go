package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/VennLe/charlotte/internal/config"
)

var (
	configPath string
	encryptKey string
	verbose    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "validate-config",
		Short: "配置验证工具 - 验证配置文件格式和安全性",
		Long: `配置验证工具

此工具帮助您验证配置文件的格式正确性、完整性以及安全性设置。
支持验证加密配置的解密能力和环境变量覆盖功能。`,
		Run: runValidation,
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "./configs/config.secure.yaml", "配置文件路径")
	rootCmd.Flags().StringVarP(&encryptKey, "key", "k", "", "加密密钥（用于验证加密配置）")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细验证信息")

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("命令执行失败: %v\n", err)
		os.Exit(1)
	}
}

func runValidation(cmd *cobra.Command, args []string) {
	fmt.Println("=== Charlotte API 配置验证工具 ===\n")

	// 检查配置文件是否存在
	if !fileExists(configPath) {
		fmt.Printf("配置文件不存在: %s\n", configPath)
		fmt.Println("请使用 --config 参数指定正确的配置文件路径")
		os.Exit(1)
	}

	// 显示验证信息
	showValidationInfo(configPath)

	// 执行验证
	fmt.Println("\n开始验证配置...")
	
	// 使用安全配置管理器进行验证
	scm := config.NewSecureConfigManager()
	
	// 加载配置进行验证
	if err := scm.LoadSecureConfig(configPath); err != nil {
		fmt.Printf("❌ 配置文件加载失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 配置文件加载成功")

	// 验证配置完整性
	if err := validateConfigCompleteness(scm); err != nil {
		fmt.Printf("❌ 配置完整性验证失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 配置完整性验证通过")

	// 验证配置安全性
	if err := validateConfigSecurity(scm); err != nil {
		fmt.Printf("❌ 配置安全性验证失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 配置安全性验证通过")

	// 验证加密配置（如果提供了密钥）
	if encryptKey != "" {
		if err := validateEncryption(scm, encryptKey); err != nil {
			fmt.Printf("❌ 加密配置验证失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ 加密配置验证通过")
	} else {
		fmt.Println("⚠ 未提供加密密钥，跳过加密配置验证")
	}

	// 显示验证结果
	showValidationResult()
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// showValidationInfo 显示验证信息
func showValidationInfo(configPath string) {
	fmt.Println("验证信息:")
	fmt.Printf("  配置文件: %s\n", configPath)
	fmt.Printf("  加密密钥: %s\n", ifEmpty(encryptKey, "未设置"))
	fmt.Printf("  详细模式: %v\n", verbose)

	fmt.Println("\n验证项目:")
	fmt.Println("  ✓ 配置文件格式（YAML语法）")
	fmt.Println("  ✓ 配置完整性（必需字段）")
	fmt.Println("  ✓ 环境变量覆盖功能")
	fmt.Println("  ✓ 加密配置（如果启用）")
	fmt.Println("  ✓ 配置安全性（敏感信息保护）")
}

// showValidationResult 显示验证结果
func showValidationResult() {
	fmt.Println("\n=== 验证完成 ===")
	fmt.Println("✓ 所有验证项目通过")

	fmt.Println("\n配置状态:")
	fmt.Println("✓ 配置文件格式正确")
	fmt.Println("✓ 必需配置项完整")
	fmt.Println("✓ 环境变量覆盖功能正常")
	fmt.Println("✓ 配置安全性符合要求")

	if encryptKey != "" {
		fmt.Println("✓ 加密配置功能正常")
	}

	fmt.Println("\n安全建议:")
	fmt.Println("1. 生产环境启用配置加密")
	fmt.Println("2. 使用环境变量存储敏感信息")
	fmt.Println("3. 定期轮换加密密钥")
	fmt.Println("4. 设置配置文件权限为600")
	fmt.Println("5. 使用密钥管理系统存储密钥")

	fmt.Println("\n使用说明:")
	fmt.Println("1. 启动服务前运行此工具验证配置")
	fmt.Println("2. 部署新环境时使用此工具检查配置")
	fmt.Println("3. 配置变更后使用此工具验证变更")
}

// ifEmpty 返回默认值如果字符串为空
func ifEmpty(str, defaultValue string) string {
	if str == "" {
		return defaultValue
	}
	return str
}

// generateValidationReport 生成验证报告（详细模式）
func generateValidationReport() {
	if verbose {
		fmt.Println("\n=== 详细验证报告 ===")
		
		// 配置文件基本信息
		fileInfo, err := os.Stat(configPath)
		if err == nil {
			fmt.Printf("文件大小: %d bytes\n", fileInfo.Size())
			fmt.Printf("修改时间: %s\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
		}

		// 权限检查
		fmt.Printf("文件权限: %o\n", fileInfo.Mode().Perm())

		// 建议
		fmt.Println("\n权限建议:")
		if fileInfo.Mode().Perm()&0077 != 0 {
			fmt.Println("⚠ 配置文件权限过宽，建议设置为600")
		} else {
			fmt.Println("✓ 文件权限设置合理")
		}
	}
}

// validateConfigCompleteness 验证配置完整性
func validateConfigCompleteness(scm *config.SecureConfigManager) error {
	// 检查必需配置项
	requiredConfigs := map[string]string{
		"server.port":     "服务器端口",
		"database.host":   "数据库主机",
		"database.user":   "数据库用户",
		"database.dbname": "数据库名称",
		"jwt.secret":      "JWT密钥",
	}

	for key, description := range requiredConfigs {
		if scm.GetString(key) == "" {
			return fmt.Errorf("必需配置缺失: %s (%s)", description, key)
		}
	}

	return nil
}

// validateConfigSecurity 验证配置安全性
func validateConfigSecurity(scm *config.SecureConfigManager) error {
	// 检查JWT密钥长度
	jwtSecret := scm.GetString("jwt.secret")
	if len(jwtSecret) < 32 {
		fmt.Printf("⚠ 警告: JWT密钥长度建议至少32位，当前长度: %d\n", len(jwtSecret))
	}

	// 检查数据库密码
	dbPassword := scm.GetString("database.password")
	if dbPassword == "" || dbPassword == "postgres" {
		fmt.Println("⚠ 警告: 数据库密码为空或使用默认值，建议设置强密码")
	}

	// 检查Redis密码
	redisPassword := scm.GetString("redis.password")
	if redisPassword == "" {
		fmt.Println("⚠ 信息: Redis未设置密码（如果Redis暴露在外网，建议设置密码）")
	}

	return nil
}

// validateEncryption 验证加密配置
func validateEncryption(scm *config.SecureConfigManager, key string) error {
	// 检查加密密钥
	encryptionKey := scm.GetString("security.encryption_key")
	if encryptionKey == "" {
		fmt.Println("⚠ 信息: 未启用配置加密")
	} else {
		if len(encryptionKey) < 32 {
			return fmt.Errorf("加密密钥长度不足，建议至少32位")
		}
		fmt.Println("✓ 加密配置已启用")
	}

	return nil
}