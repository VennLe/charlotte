package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/VennLe/charlotte/pkg/logger"
)

var Global *Config

type Config struct {
	Server      ServerConfig      `mapstructure:"server" json:"server"`
	Database    DatabaseConfig    `mapstructure:"database" json:"database"`
	Redis       RedisConfig       `mapstructure:"redis" json:"redis"`
	Kafka       KafkaConfig       `mapstructure:"kafka" json:"kafka"`
	Log         logger.Config     `mapstructure:"log" json:"log"`
	JWT         JWTConfig         `mapstructure:"jwt" json:"jwt"`
	Migrate     MigrateConfig     `mapstructure:"migrate" json:"migrate"`
	Performance PerformanceConfig `mapstructure:"performance" json:"performance"`
	Health      HealthConfig      `mapstructure:"health" json:"health"`
	Security    SecurityConfig    `mapstructure:"security" json:"security"`
	Monitoring  MonitoringConfig  `mapstructure:"monitoring" json:"monitoring"`
	DevTools    DevToolsConfig    `mapstructure:"devtools" json:"devtools"`
}

type PerformanceConfig struct {
	RequestTimeout  int `mapstructure:"request_timeout" json:"request_timeout"`
	ResponseTimeout int `mapstructure:"response_timeout" json:"response_timeout"`
	MaxRequestSize  int `mapstructure:"max_request_size" json:"max_request_size"`
	RateLimit       int `mapstructure:"rate_limit" json:"rate_limit"`
}

type HealthConfig struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled"`
	Path     string `mapstructure:"path" json:"path"`
	Interval int    `mapstructure:"interval" json:"interval"`
	Timeout  int    `mapstructure:"timeout" json:"timeout"`
}

type SecurityConfig struct {
	CORSEnabled        bool     `mapstructure:"cors_enabled" json:"cors_enabled"`
	CORSOrigins        []string `mapstructure:"cors_origins" json:"cors_origins"`
	RateLimitEnabled   bool     `mapstructure:"rate_limit_enabled" json:"rate_limit_enabled"`
	RateLimitPerMinute int      `mapstructure:"rate_limit_per_minute" json:"rate_limit_per_minute"`
}

type MonitoringConfig struct {
	MetricsEnabled     bool   `mapstructure:"metrics_enabled" json:"metrics_enabled"`
	MetricsPath        string `mapstructure:"metrics_path" json:"metrics_path"`
	TracingEnabled     bool   `mapstructure:"tracing_enabled" json:"tracing_enabled"`
	HealthCheckEnabled bool   `mapstructure:"health_check_enabled" json:"health_check_enabled"`
}

type DevToolsConfig struct {
	PprofEnabled   bool   `mapstructure:"pprof_enabled" json:"pprof_enabled"`
	PprofPort      int    `mapstructure:"pprof_port" json:"pprof_port"`
	MetricsEnabled bool   `mapstructure:"metrics_enabled" json:"metrics_enabled"`
	MetricsPath    string `mapstructure:"metrics_path" json:"metrics_path"`
}

type MigrateConfig struct {
	Enabled       bool     `mapstructure:"enabled" json:"enabled"`
	AutoMigrate   bool     `mapstructure:"auto_migrate" json:"auto_migrate"`
	DropTables    bool     `mapstructure:"drop_tables" json:"drop_tables"`
	Models        []string `mapstructure:"models" json:"models"`
	CreateIndexes bool     `mapstructure:"create_indexes" json:"create_indexes"`
	Verbose       bool     `mapstructure:"verbose" json:"verbose"`
}

type ServerConfig struct {
	Name string `mapstructure:"name" json:"name"`
	Port string `mapstructure:"port" json:"port"`
	Mode string `mapstructure:"mode" json:"mode"` // debug/release
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host" json:"host"`
	Port         string `mapstructure:"port" json:"port"`
	User         string `mapstructure:"user" json:"user"`
	Password     string `mapstructure:"password" json:"password"`
	DBName       string `mapstructure:"dbname" json:"dbname"`
	MaxOpenConns int    `mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" json:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host"`
	Port     string `mapstructure:"port" json:"port"`
	Password string `mapstructure:"password" json:"password"`
	DB       int    `mapstructure:"db" json:"db"`
	PoolSize int    `mapstructure:"pool_size" json:"pool_size"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers" json:"brokers"`
	Topic   string   `mapstructure:"topic" json:"topic"`
	GroupID string   `mapstructure:"group_id" json:"group_id"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret" json:"secret"`
	Expire int    `mapstructure:"expire" json:"expire"` // 小时
}

// Load 加载配置（兼容旧版本，推荐使用LoadConfig）
func Load(cfgFile string) {
	LoadConfig(cfgFile)
}

// LoadConfig 加载配置
func LoadConfig(cfgFile string) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 环境变量支持
	v.AutomaticEnv()
	v.SetEnvPrefix("CHARLOTTE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 配置文件加载
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/charlotte")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Warn("配置文件未找到，使用默认配置和环境变量")
		} else {
			log.Fatalf("配置文件读取失败: %v", err)
		}
	}

	Global = &Config{}
	if err := v.Unmarshal(Global); err != nil {
		log.Fatalf("配置解析失败: %v", err)
	}

	// 监听配置变化
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		if err := v.Unmarshal(Global); err != nil {
			logger.Error("配置热更新失败", zap.Error(err))
			return
		}
		logger.Info("配置已热更新", zap.String("file", e.Name))
	})

	logger.Info("配置加载成功", zap.String("file", v.ConfigFileUsed()))
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.name", "charlotte-api")
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "debug")

	// 数据库默认配置
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.dbname", "charlotte")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 3600)
	v.SetDefault("database.conn_max_idle_time", 1800)

	// Redis默认配置
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", "6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.max_conn_age", 0)
	v.SetDefault("redis.pool_timeout", 4)

	// Kafka默认配置
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topic", "user-events")
	v.SetDefault("kafka.group_id", "charlotte-group")
	v.SetDefault("kafka.auto_offset_reset", "latest")
	v.SetDefault("kafka.session_timeout", 10000)
	v.SetDefault("kafka.heartbeat_interval", 3000)
	v.SetDefault("kafka.max_poll_interval", 300000)

	// JWT默认配置
	v.SetDefault("jwt.secret", "your-jwt-secret-key-here")
	v.SetDefault("jwt.expire", 24)
	v.SetDefault("jwt.issuer", "charlotte-api")
	v.SetDefault("jwt.audience", "charlotte-users")

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.encoding", "json")
	v.SetDefault("log.output_paths", []string{"stdout"})
	v.SetDefault("log.error_output_paths", []string{"stderr"})
	v.SetDefault("log.development", false)

	// 迁移默认配置
	v.SetDefault("migrate.enabled", true)
	v.SetDefault("migrate.auto_migrate", true)
	v.SetDefault("migrate.drop_tables", false)
	v.SetDefault("migrate.create_indexes", true)
	v.SetDefault("migrate.verbose", true)

	// 性能配置默认值
	v.SetDefault("performance.request_timeout", 30)
	v.SetDefault("performance.response_timeout", 30)
	v.SetDefault("performance.max_request_size", 10485760)
	v.SetDefault("performance.rate_limit", 1000)

	// 健康检查默认值
	v.SetDefault("health.enabled", true)
	v.SetDefault("health.path", "/health")
	v.SetDefault("health.interval", 30)
	v.SetDefault("health.timeout", 5)

	// 安全配置默认值
	v.SetDefault("security.cors_enabled", true)
	v.SetDefault("security.cors_origins", []string{"*"})
	v.SetDefault("security.rate_limit_enabled", true)
	v.SetDefault("security.rate_limit_per_minute", 100)

	// 监控配置默认值
	v.SetDefault("monitoring.metrics_enabled", true)
	v.SetDefault("monitoring.metrics_path", "/metrics")
	v.SetDefault("monitoring.tracing_enabled", false)
	v.SetDefault("monitoring.health_check_enabled", true)

	// 开发工具默认值
	v.SetDefault("devtools.pprof_enabled", false)
	v.SetDefault("devtools.pprof_port", 6060)
	v.SetDefault("devtools.metrics_enabled", false)
	v.SetDefault("devtools.metrics_path", "/metrics")
}

func Show() {
	fmt.Printf("%+v\n", Global)
}

// Validate 验证配置的完整性
func Validate() error {
	if Global == nil {
		return fmt.Errorf("配置未加载")
	}

	// 验证服务器配置
	if Global.Server.Name == "" {
		return fmt.Errorf("服务器名称不能为空")
	}
	if Global.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}

	// 验证数据库配置
	if Global.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}
	if Global.Database.User == "" {
		return fmt.Errorf("数据库用户不能为空")
	}
	if Global.Database.DBName == "" {
		return fmt.Errorf("数据库名称不能为空")
	}

	// 验证Redis配置
	if Global.Redis.Host == "" {
		return fmt.Errorf("Redis主机不能为空")
	}

	// 验证JWT配置
	if Global.JWT.Secret == "" {
		return fmt.Errorf("JWT密钥不能为空")
	}
	if len(Global.JWT.Secret) < 32 {
		logger.Warn("JWT密钥长度建议至少32位，当前密钥安全性较低")
	}

	// 验证Kafka配置
	if len(Global.Kafka.Brokers) == 0 {
		logger.Warn("Kafka代理未配置，Kafka功能将不可用")
	}

	logger.Info("配置验证通过")
	return nil
}

// GetConfigSummary 获取配置摘要（用于日志输出）
func GetConfigSummary() string {
	if Global == nil {
		return "配置未加载"
	}

	return fmt.Sprintf(`
配置摘要:
  Server:   %s:%s (%s)
  Database: %s@%s:%s/%s
  Redis:    %s:%s
  Kafka:    %d brokers
  Nacos:    %s:%d
  JWT:      %d小时过期
`,
		Global.Server.Name, Global.Server.Port, Global.Server.Mode,
		Global.Database.User, Global.Database.Host, Global.Database.Port, Global.Database.DBName,
		Global.Redis.Host, Global.Redis.Port,
		len(Global.Kafka.Brokers),
		Global.JWT.Expire)
}
