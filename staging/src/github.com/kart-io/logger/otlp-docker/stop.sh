#!/bin/bash

# OTLP Stack 停止脚本
# 用途：停止并清理 OTLP 技术栈
# 作者：Claude Code Assistant
# 版本：1.0.0

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 停止服务
stop_services() {
    log_info "停止OTLP技术栈服务..."
    
    # 确保在正确的目录
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cd "$SCRIPT_DIR"
    
    if [[ -f "docker-compose.yml" ]]; then
        docker-compose down
        log_info "✅ Docker Compose服务已停止"
    else
        log_warn "未找到docker-compose.yml文件"
    fi
}

# 清理资源
cleanup_resources() {
    local remove_volumes=$1
    local remove_images=$2
    
    log_info "清理OTLP相关资源..."
    
    # 清理容器
    containers=(
        "kart-otel-agent"
        "kart-otel-collector" 
        "kart-victorialogs"
        "kart-jaeger"
        "kart-prometheus"
    )
    
    for container in "${containers[@]}"; do
        if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
            log_debug "删除容器: $container"
            docker rm -f "$container" 2>/dev/null || true
        fi
    done
    
    # 清理网络
    if docker network ls --format '{{.Name}}' | grep -q "^kart-otlp-network$"; then
        log_debug "删除网络: kart-otlp-network"
        docker network rm kart-otlp-network 2>/dev/null || true
    fi
    
    # 清理数据卷（如果请求）
    if [[ "$remove_volumes" == true ]]; then
        log_warn "清理数据卷（这将删除所有存储的数据）..."
        
        volumes=(
            "otlp-docker_otel-agent-data"
            "otlp-docker_otel-collector-data"
            "otlp-docker_victorialogs-data"
            "otlp-docker_jaeger-data"
            "otlp-docker_prometheus-data"
        )
        
        for volume in "${volumes[@]}"; do
            if docker volume ls --format '{{.Name}}' | grep -q "^${volume}$"; then
                log_debug "删除数据卷: $volume"
                docker volume rm "$volume" 2>/dev/null || true
            fi
        done
        
        log_info "✅ 数据卷清理完成"
    fi
    
    # 清理镜像（如果请求）
    if [[ "$remove_images" == true ]]; then
        log_warn "清理Docker镜像..."
        
        images=(
            "otel/opentelemetry-collector-contrib:0.132.0"
            "victoriametrics/victoria-logs:v1.28.0-victorialogs" 
            "jaegertracing/all-in-one:1.57"
            "prom/prometheus:v2.51.0"
        )
        
        for image in "${images[@]}"; do
            if docker images --format '{{.Repository}}:{{.Tag}}' | grep -q "^${image}$"; then
                log_debug "删除镜像: $image"
                docker rmi "$image" 2>/dev/null || true
            fi
        done
        
        log_info "✅ 镜像清理完成"
    fi
    
    log_info "✅ 资源清理完成"
}

# 显示状态
show_status() {
    log_info "检查剩余的OTLP相关资源..."
    
    echo
    log_debug "运行中的容器:"
    if docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E "(kart-|otel|victoria|jaeger|prometheus)" || true; then
        echo "  (无相关容器运行中)"
    fi
    
    echo
    log_debug "OTLP相关数据卷:"
    if docker volume ls --format 'table {{.Name}}\t{{.Driver}}' | grep -E "(otel|victoria|jaeger|prometheus)" || true; then
        echo "  (无相关数据卷)"
    fi
    
    echo
    log_debug "OTLP相关网络:"
    if docker network ls --format 'table {{.Name}}\t{{.Driver}}' | grep -E "(kart-|otlp)" || true; then
        echo "  (无相关网络)"
    fi
}

# 显示使用说明
show_usage() {
    echo "OTLP Stack 停止脚本"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help          显示此帮助信息"
    echo "  -v, --volumes       同时删除数据卷（会丢失数据）"
    echo "  -i, --images        同时删除Docker镜像"
    echo "  --all               删除所有资源（容器、卷、镜像）"
    echo "  --force             强制删除，不询问确认"
    echo
    echo "示例:"
    echo "  $0                  # 仅停止服务"
    echo "  $0 --volumes        # 停止服务并删除数据"
    echo "  $0 --all            # 完全清理"
    echo
}

# 确认操作
confirm_action() {
    local action=$1
    local force=$2
    
    if [[ "$force" != true ]]; then
        log_warn "$action"
        read -p "确认执行此操作吗？ (y/N): " -n 1 -r
        echo
        
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            exit 0
        fi
    fi
}

# 主函数
main() {
    local remove_volumes=false
    local remove_images=false
    local force=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--volumes)
                remove_volumes=true
                shift
                ;;
            -i|--images)
                remove_images=true
                shift
                ;;
            --all)
                remove_volumes=true
                remove_images=true
                shift
                ;;
            --force)
                force=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    echo "🛑 OTLP Stack 停止脚本启动"
    echo "================================================"
    
    # 确认危险操作
    if [[ "$remove_volumes" == true ]] && [[ "$remove_images" == true ]]; then
        confirm_action "这将删除所有OTLP相关的容器、数据卷和镜像，所有数据将丢失！" "$force"
    elif [[ "$remove_volumes" == true ]]; then
        confirm_action "这将删除所有数据卷，存储的日志、指标和追踪数据将丢失！" "$force"
    elif [[ "$remove_images" == true ]]; then
        confirm_action "这将删除OTLP相关的Docker镜像，下次启动需要重新拉取。" "$force"
    fi
    
    # 执行停止步骤
    stop_services
    cleanup_resources "$remove_volumes" "$remove_images"
    show_status
    
    echo
    echo "================================================"
    log_info "🏁 OTLP技术栈停止完成！"
    echo
    
    if [[ "$remove_volumes" == true ]] || [[ "$remove_images" == true ]]; then
        log_info "资源清理完成。重新部署请运行: ./deploy.sh"
    else
        log_info "数据和镜像已保留。重新启动请运行: docker-compose up -d"
    fi
}

# 执行主函数
main "$@"