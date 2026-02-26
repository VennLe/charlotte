package initialize

import (
	"context"
	"fmt"
	"github.com/VennLe/charlotte/internal/model"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"

	"github.com/VennLe/charlotte/internal/config"
	"github.com/VennLe/charlotte/pkg/logger"
)

var DB *gorm.DB

func InitGorm() error {
	cfg := config.Global.Database

	// 根据数据库类型生成DSN
	dsn, err := generateDSN(cfg)
	if err != nil {
		return fmt.Errorf("生成DSN失败: %w", err)
	}

	// 根据环境设置日志级别
	var logLevel gLogger.LogLevel
	if config.Global.Server.Mode == "debug" {
		logLevel = gLogger.Info
	} else {
		logLevel = gLogger.Error
	}

	// 根据数据库类型创建连接
	var db *gorm.DB
	switch cfg.Type {
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gLogger.Default.LogMode(logLevel),
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: gLogger.Default.LogMode(logLevel),
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		})
	case "postgres", "": // 默认使用postgres
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gLogger.Default.LogMode(logLevel),
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		})
	default:
		return fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}

	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 设置连接池参数
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(10)
	}

	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
	} else {
		sqlDB.SetConnMaxIdleTime(30 * time.Minute)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库 ping 失败: %w", err)
	}

	DB = db
	logger.Info("数据库连接成功",
		zap.String("type", cfg.Type),
		zap.String("host", cfg.Host),
		zap.String("dbname", cfg.DBName))
	return nil
}

// generateDSN 根据数据库类型生成DSN字符串
func generateDSN(cfg config.DatabaseConfig) (string, error) {
	switch cfg.Type {
	case "mysql":
		// MySQL DSN格式: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
		charset := cfg.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		parseTime := cfg.ParseTime
		if !parseTime {
			parseTime = true
		}
		loc := cfg.Loc
		if loc == "" {
			loc = "Local"
		}
		
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=%t&loc=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, charset, parseTime, loc), nil
		
	case "sqlite":
		// SQLite DSN格式: file:path?cache=shared&mode=rwc
		path := cfg.SQLitePath
		if path == "" {
			path = "./data/charlotte.db"
		}
		return fmt.Sprintf("file:%s?cache=shared&mode=rwc", path), nil
		
	case "postgres", "":
		// PostgreSQL DSN格式: host= user= password= dbname= port= sslmode=disable TimeZone=Asia/Shanghai
		return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port), nil
		
	default:
		return "", fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}
}

// Migrate 执行数据库迁移
func Migrate() error {
	if DB == nil {
		if err := InitGorm(); err != nil {
			return err
		}
	}

	// 检查是否启用迁移
	if !config.Global.Migrate.Enabled {
		logger.Info("数据库迁移已禁用，跳过迁移")
		return nil
	}

	logger.Info("开始执行数据库迁移...",
		zap.Bool("auto_migrate", config.Global.Migrate.AutoMigrate),
		zap.Bool("drop_tables", config.Global.Migrate.DropTables),
		zap.Bool("verbose", config.Global.Migrate.Verbose))

	// 自动迁移表结构
	if config.Global.Migrate.AutoMigrate {
		models := []interface{}{
			&model.User{},
			// 在这里添加其他模型...
		}

		err := DB.AutoMigrate(models...)
		if err != nil {
			logger.Error("数据库迁移失败", zap.Error(err))
			return err
		}

		logger.Info("数据库自动迁移完成", zap.Int("models_count", len(models)))
	}

	// 如果需要创建索引
	if config.Global.Migrate.CreateIndexes {
		if err := createIndexes(); err != nil {
			logger.Warn("创建索引失败", zap.Error(err))
		}
	}

	logger.Info("数据库迁移完成")
	return nil
}

// createIndexes 创建数据库索引
func createIndexes() error {
	// 用户表索引
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`).Error; err != nil {
		return err
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`).Error; err != nil {
		return err
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)`).Error; err != nil {
		return err
	}

	logger.Info("数据库索引创建完成")
	return nil
}