package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/VennLe/charlotte/internal/config"
)

var (
	sourcePath string
	targetPath string
	encrypt    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "migrate-config",
		Short: "配置迁移工具 - 从旧配置系统迁移到新安全配置系统",
		Long: `配置迁移工具

此工具帮助您从旧的.env和多个YAML配置文件迁移到统一的安全配置系统。
新系统支持配置加密、环境变量优先级和配置热更新。`,
		Run: runMigration,
	}

	rootCmd.Flags().StringVarP(&sourcePath, "source", "s", ".", "源配置路径（包含.env和configs目录）")
	rootCmd.Flags().StringVarP(&targetPath, "target", "t", "./configs/config.secure.yaml", "目标配置文件路径")
	rootCmd.Flags().BoolVarP(&encrypt, "encrypt", "e", false, "是否加密敏感配置")

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("命令执行失败: %v\n", err)
		os.Exit(1)
	}
}

func runMigration(cmd *cobra.Command, args []string) {
	fmt.Println("=== Charlotte API 配置迁移工具 ===\n")

	// 检查源路径
	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		fmt.Printf("源路径解析失败: %v\n", err)
		os.Exit(1)
	}

	// 检查源文件是否存在
	if !checkSourceFiles(absSourcePath) {
		fmt.Println("未找到有效的源配置文件，请检查源路径。")
		fmt.Println("支持的源文件:")
		fmt.Println("  - .env (环境变量文件)")
		fmt.Println("  - configs/config.yaml (主配置文件)")
		fmt.Println("  - configs/config.dev.yaml (开发环境配置)")
		fmt.Println("  - configs/config.production.yaml (生产环境配置)")
		os.Exit(1)
	}

	// 创建迁移实例
	migration := config.NewConfigMigration(absSourcePath, targetPath)

	// 显示迁移信息
	showMigrationInfo(absSourcePath, targetPath)

	// 执行迁移
	fmt.Println("\n开始迁移配置...")
	if err := migration.MigrateFromEnvAndYaml(); err != nil {
		fmt.Printf("迁移失败: %v\n", err)
		os.Exit(1)
	}

	// 显示迁移结果
	showMigrationResult(targetPath)
}

// checkSourceFiles 检查源文件是否存在
func checkSourceFiles(sourcePath string) bool {
	files := []string{
		filepath.Join(sourcePath, ".env"),
		filepath.Join(sourcePath, "configs", "config.yaml"),
		filepath.Join(sourcePath, "configs", "config.dev.yaml"),
		filepath.Join(sourcePath, "configs", "config.production.yaml"),
	}

	found := false
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("✓ 找到源文件: %s\n", file)
			found = true
		}
	}

	return found
}

// showMigrationInfo 显示迁移信息
func showMigrationInfo(sourcePath, targetPath string) {
	fmt.Println("\n迁移信息:")
	fmt.Printf("  源路径: %s\n", sourcePath)
	fmt.Printf("  目标文件: %s\n", targetPath)
	fmt.Printf("  加密敏感配置: %v\n", encrypt)

	fmt.Println("\n迁移内容:")
	fmt.Println("  - 环境变量配置 (.env)")
	fmt.Println("  - YAML配置文件 (configs/)")
	fmt.Println("  - 数据库连接信息")
	fmt.Println("  - Redis连接信息")
	fmt.Println("  - JWT认证配置")
	fmt.Println("  - 服务器配置")
	fmt.Println("  - 日志配置")
	fmt.Println("  - 性能配置")
}

// showMigrationResult 显示迁移结果
func showMigrationResult(targetPath string) {
	fmt.Println("\n=== 迁移完成 ===")
	fmt.Printf("新配置文件已生成: %s\n", targetPath)

	fmt.Println("\n下一步操作:")
	fmt.Println("1. 检查新配置文件内容")
	fmt.Println("2. 设置环境变量（可选）:")
	fmt.Println("   export CHARLOTTE_ENCRYPTION_KEY=your-encryption-key")
	fmt.Println("   export CHARLOTTE_DB_PASSWORD=your-db-password")
	fmt.Println("   export CHARLOTTE_JWT_SECRET=your-jwt-secret")
	fmt.Println("3. 更新应用代码使用新配置系统")
	fmt.Println("4. 删除旧的配置文件（备份后）")

	fmt.Println("\n安全建议:")
	fmt.Println("✓ 使用环境变量存储敏感信息")
	fmt.Println("✓ 设置配置加密密钥增强安全性")
	fmt.Println("✓ 定期轮换加密密钥")
	fmt.Println("✓ 使用密钥管理系统存储密钥")

	fmt.Println("\n新配置系统特性:")
	fmt.Println("✓ 统一配置管理")
	fmt.Println("✓ 环境变量优先级")
	fmt.Println("✓ 敏感配置加密")
	fmt.Println("✓ 配置热更新")
	fmt.Println("✓ 配置验证")
	fmt.Println("✓ 多环境支持")
}

// generateConfigTemplate 生成配置模板（备用功能）
func generateConfigTemplate() {
	fmt.Println("\n生成配置模板...")
	
	template := `# Charlotte API 服务安全配置
# 此文件支持环境变量覆盖和配置加密

server:
  name: ${CHARLOTTE_SERVER_NAME:-charlotte-api}
  port: ${CHARLOTTE_SERVER_PORT:-8080}
  mode: ${CHARLOTTE_SERVER_MODE:-debug}

database:
  host: ${CHARLOTTE_DB_HOST:-localhost}
  port: ${CHARLOTTE_DB_PORT:-5432}
  user: ${CHARLOTTE_DB_USER:-postgres}
  password: ${CHARLOTTE_DB_PASSWORD:-postgres}
  dbname: ${CHARLOTTE_DB_NAME:-charlotte}

redis:
  host: ${CHARLOTTE_REDIS_HOST:-localhost}
  port: ${CHARLOTTE_REDIS_PORT:-6379}
  password: ${CHARLOTTE_REDIS_PASSWORD:-}
  db: ${CHARLOTTE_REDIS_DB:-0}

jwt:
  secret: ${CHARLOTTE_JWT_SECRET:-your-jwt-secret-key}
  expire: ${CHARLOTTE_JWT_EXPIRE:-24}

# 启用配置加密（可选）
security:
  encryption_key: ${CHARLOTTE_ENCRYPTION_KEY:-}
`

	// 保存模板文件
	templatePath := "./configs/config.template.yaml"
	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		fmt.Printf("生成模板失败: %v\n", err)
		return
	}

	fmt.Printf("配置模板已生成: %s\n", templatePath)
}