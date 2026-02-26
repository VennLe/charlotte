#!/bin/bash

# Charlotte API 配置部署脚本
# 自动化配置迁移、验证和部署过程

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
Charlotte API 配置部署脚本

用法: $0 [选项]

选项:
    -h, --help          显示此帮助信息
    -m, --migrate       执行配置迁移
    -v, --validate      执行配置验证
    -d, --deploy        执行完整部署
    -e, --encrypt       启用配置加密
    -c, --config FILE   指定配置文件路径
    -s, --source DIR    指定源配置目录
    -t, --target FILE   指定目标配置文件
    --dry-run           干运行模式（不实际执行）
    --force             强制覆盖现有文件

示例:
    # 迁移配置
    $0 --migrate
    
    # 验证配置
    $0 --validate
    
    # 完整部署
    $0 --deploy --encrypt
    
    # 自定义路径
    $0 --migrate --source ./old-configs --target ./configs/new-config.yaml

EOF
}

# 参数解析
MIGRATE=false
VALIDATE=false
DEPLOY=false
ENCRYPT=false
DRY_RUN=false
FORCE=false
CONFIG_FILE=""
SOURCE_DIR="."
TARGET_FILE="./configs/config.secure.yaml"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -m|--migrate)
            MIGRATE=true
            shift
            ;;
        -v|--validate)
            VALIDATE=true
            shift
            ;;
        -d|--deploy)
            DEPLOY=true
            shift
            ;;
        -e|--encrypt)
            ENCRYPT=true
            shift
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -s|--source)
            SOURCE_DIR="$2"
            shift 2
            ;;
        -t|--target)
            TARGET_FILE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        *)
            log_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# 如果没有指定操作，显示帮助
if [[ $MIGRATE == false && $VALIDATE == false && $DEPLOY == false ]]; then
    show_help
    exit 0
fi

# 检查必要工具
check_dependencies() {
    log_info "检查依赖工具..."
    
    local missing_tools=()
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        missing_tools+=("Go")
    fi
    
    # 检查git
    if ! command -v git &> /dev/null; then
        missing_tools+=("Git")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "缺少必要工具: ${missing_tools[*]}"
        exit 1
    fi
    
    log_success "依赖工具检查通过"
}

# 检查项目结构
check_project_structure() {
    log_info "检查项目结构..."
    
    local required_dirs=("cmd" "internal" "configs")
    local required_files=("go.mod" "main.go")
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_error "缺少必要目录: $dir"
            exit 1
        fi
    done
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log_error "缺少必要文件: $file"
            exit 1
        fi
    done
    
    log_success "项目结构检查通过"
}

# 备份现有配置
backup_configs() {
    log_info "备份现有配置..."
    
    local backup_dir="backup/configs_$(date +%Y%m%d_%H%M%S)"
    
    if [[ $DRY_RUN == true ]]; then
        log_info "[DRY RUN] 将创建备份目录: $backup_dir"
        return
    fi
    
    mkdir -p "$backup_dir"
    
    # 备份配置文件
    if [[ -f ".env" ]]; then
        cp ".env" "$backup_dir/"
    fi
    
    if [[ -d "configs" ]]; then
        cp -r "configs" "$backup_dir/"
    fi
    
    log_success "配置已备份到: $backup_dir"
}

# 执行配置迁移
migrate_configs() {
    log_info "执行配置迁移..."
    
    local migrate_cmd="go run cmd/migrate_config.go"
    
    if [[ -n "$SOURCE_DIR" ]]; then
        migrate_cmd+=" --source \"$SOURCE_DIR\""
    fi
    
    if [[ -n "$TARGET_FILE" ]]; then
        migrate_cmd+=" --target \"$TARGET_FILE\""
    fi
    
    if [[ $ENCRYPT == true ]]; then
        migrate_cmd+=" --encrypt"
    fi
    
    if [[ $DRY_RUN == true ]]; then
        log_info "[DRY RUN] 将执行: $migrate_cmd"
        return
    fi
    
    log_info "执行命令: $migrate_cmd"
    
    if eval "$migrate_cmd"; then
        log_success "配置迁移完成"
    else
        log_error "配置迁移失败"
        exit 1
    fi
}

# 验证配置
validate_configs() {
    log_info "验证配置..."
    
    local config_to_validate="$CONFIG_FILE"
    if [[ -z "$config_to_validate" ]]; then
        config_to_validate="$TARGET_FILE"
    fi
    
    if [[ ! -f "$config_to_validate" ]]; then
        log_error "配置文件不存在: $config_to_validate"
        exit 1
    fi
    
    local validate_cmd="go run cmd/validate_config.go --config \"$config_to_validate\""
    
    if [[ $ENCRYPT == true ]]; then
        validate_cmd+=" --encryption"
    fi
    
    if [[ $DRY_RUN == true ]]; then
        log_info "[DRY RUN] 将执行: $validate_cmd"
        return
    fi
    
    log_info "执行命令: $validate_cmd"
    
    if eval "$validate_cmd"; then
        log_success "配置验证通过"
    else
        log_error "配置验证失败"
        exit 1
    fi
}

# 生成环境变量模板
generate_env_template() {
    log_info "生成环境变量模板..."
    
    local env_template=".env.template"
    
    if [[ $DRY_RUN == true ]]; then
        log_info "[DRY RUN] 将生成环境变量模板: $env_template"
        return
    fi
    
    cat > "$env_template" << 'EOF'
# Charlotte API 环境变量配置模板
# 复制此文件为 .env 并填入实际值

# 服务器配置
CHARLOTTE_SERVER_NAME=charlotte-api
CHARLOTTE_SERVER_PORT=8080
CHARLOTTE_SERVER_MODE=debug

# 数据库配置
CHARLOTTE_DB_HOST=localhost
CHARLOTTE_DB_PORT=5432
CHARLOTTE_DB_USER=postgres
CHARLOTTE_DB_PASSWORD=your-secure-password
CHARLOTTE_DB_NAME=charlotte

# Redis配置
CHARLOTTE_REDIS_HOST=localhost
CHARLOTTE_REDIS_PORT=6379
CHARLOTTE_REDIS_PASSWORD=your-redis-password
CHARLOTTE_REDIS_DB=0

# JWT配置
CHARLOTTE_JWT_SECRET=your-32-byte-jwt-secret-key-here
CHARLOTTE_JWT_EXPIRE=24

# 安全配置
CHARLOTTE_ENCRYPTION_KEY=your-32-byte-encryption-key-here

# 日志配置
CHARLOTTE_LOG_LEVEL=info
CHARLOTTE_LOG_FORMAT=json

# 性能配置
CHARLOTTE_MAX_CONNECTIONS=100
CHARLOTTE_TIMEOUT=30

EOF

    log_success "环境变量模板已生成: $env_template"
}

# 设置文件权限
set_file_permissions() {
    log_info "设置文件权限..."
    
    if [[ $DRY_RUN == true ]]; then
        log_info "[DRY RUN] 将设置配置文件权限为600"
        return
    fi
    
    # 设置配置文件权限
    if [[ -f "$TARGET_FILE" ]]; then
        chmod 600 "$TARGET_FILE"
        log_success "配置文件权限已设置为600"
    fi
    
    # 设置环境文件权限
    if [[ -f ".env" ]]; then
        chmod 600 ".env"
        log_success "环境文件权限已设置为600"
    fi
}

# 显示部署摘要
show_deployment_summary() {
    log_info "=== 部署摘要 ==="
    
    echo ""
    echo "📋 部署完成:"
    echo ""
    
    if [[ $MIGRATE == true ]]; then
        echo "✅ 配置迁移完成"
        echo "   目标文件: $TARGET_FILE"
    fi
    
    if [[ $VALIDATE == true ]]; then
        echo "✅ 配置验证完成"
    fi
    
    if [[ $ENCRYPT == true ]]; then
        echo "🔒 配置加密已启用"
    fi
    
    echo ""
    echo "📝 下一步操作:"
    echo ""
    echo "1. 检查配置文件: $TARGET_FILE"
    echo "2. 设置环境变量（可选）:"
    echo "   cp .env.template .env"
    echo "   # 编辑 .env 文件填入实际值"
    echo "3. 启动服务:"
    echo "   go run main.go"
    echo ""
    echo "🔒 安全建议:"
    echo ""
    echo "• 使用强密码和密钥"
    echo "• 定期轮换加密密钥"
    echo "• 使用密钥管理系统"
    echo "• 设置文件权限为600"
    echo ""
}

# 主函数
main() {
    log_info "开始 Charlotte API 配置部署..."
    
    # 检查依赖
    check_dependencies
    
    # 检查项目结构
    check_project_structure
    
    # 备份配置
    backup_configs
    
    # 执行迁移
    if [[ $MIGRATE == true || $DEPLOY == true ]]; then
        migrate_configs
    fi
    
    # 执行验证
    if [[ $VALIDATE == true || $DEPLOY == true ]]; then
        validate_configs
    fi
    
    # 生成环境模板
    if [[ $DEPLOY == true ]]; then
        generate_env_template
    fi
    
    # 设置文件权限
    if [[ $DEPLOY == true ]]; then
        set_file_permissions
    fi
    
    # 显示摘要
    show_deployment_summary
    
    log_success "配置部署完成！"
}

# 执行主函数
main "$@"