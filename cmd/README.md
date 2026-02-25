# Charlotte API 命令结构

## 概述

Charlotte API 采用精简的命令结构，将功能模块化到不同的文件中：

- `root.go` - 根命令定义和初始化
- `run.go` - 子命令实现和业务逻辑
- `version.go` - 版本信息管理
- `main.go` - 程序入口点

## 文件说明

### root.go
- 定义根命令 `charlotte`
- 处理配置文件的加载和初始化
- 设置 GOMAXPROCS 优化并发性能
- 管理全局命令行参数

### run.go
- 实现所有子命令：
  - `start` - 启动 API 服务器
  - `migrate` - 执行数据库迁移
  - `config show` - 显示当前配置
  - `config validate` - 验证配置完整性
  - `config env` - 显示环境变量映射
  - `version` - 显示版本信息
- 包含服务器启动和优雅关闭逻辑

### version.go
- 管理版本信息结构体
- 提供多种版本信息输出格式
- 支持详细版本信息展示

### main.go
- 简洁的程序入口点
- 调用 Execute() 函数启动命令行应用

## 使用方法

### 查看帮助
```bash
# 查看所有命令
./charlotte --help

# 查看特定命令帮助
./charlotte start --help
./charlotte config --help
```

### 启动服务
```bash
# 使用默认配置
./charlotte start

# 指定配置文件
./charlotte start --config configs/config.production.yaml

# 使用环境变量覆盖配置
export CHARLOTTE_DATABASE_PASSWORD=your_password
./charlotte start
```

### 配置管理
```bash
# 显示当前配置
./charlotte config show

# 验证配置完整性
./charlotte config validate

# 查看环境变量映射
./charlotte config env
```

### 数据库操作
```bash
# 执行数据库迁移
./charlotte migrate
```

### 版本信息
```bash
# 显示版本信息
./charlotte version
```

## 命令结构优势

1. **模块化设计** - 每个文件职责单一，便于维护
2. **易于扩展** - 新增命令只需在 run.go 中添加
3. **代码复用** - 版本信息独立管理，可在多处使用
4. **结构清晰** - 根命令、子命令、业务逻辑分离

## 开发指南

### 添加新命令

1. 在 `run.go` 的 `init()` 函数中添加命令：
```go
rootCmd.AddCommand(newCmd)
```

2. 定义新命令：
```go
var newCmd = &cobra.Command{
    Use:   "new",
    Short: "新命令描述",
    Run: func(cmd *cobra.Command, args []string) {
        // 命令逻辑
    },
}
```

### 修改版本信息

在 `root.go` 中修改变量：
```go
var (
    Version   = "1.0.0"
    BuildTime = "2024-01-01T00:00:00Z"
)
```

### 添加命令行参数

在 `root.go` 的 `init()` 函数中添加：
```go
rootCmd.PersistentFlags().StringVarP(&newParam, "param", "p", "", "参数描述")
```