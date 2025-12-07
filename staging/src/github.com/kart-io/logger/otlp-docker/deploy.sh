#!/bin/bash

# OTLP Stack 部署脚本
# 用途：部署完整的 Application → Agent → Collector → VictoriaLogs 链路
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

# 检查依赖
check_dependencies() {
    log_info "检查依赖项..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在PATH中"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装或不在PATH中"
        exit 1
    fi
    
    log_info "✅ 依赖项检查通过"
}

# 清理旧的容器和网络
cleanup_old() {
    log_info "清理旧的OTLP容器和网络..."
    
    # 停止并删除旧的容器
    for container in kart-otel-agent kart-otel-collector kart-victorialogs kart-jaeger kart-prometheus proj-otel-agent proj-otel-collector proj-victorialogs proj-jaeger; do
        if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
            log_debug "删除容器: $container"
            docker rm -f $container 2>/dev/null || true
        fi
    done
    
    # 删除旧的网络
    for network in kart-otlp-network proj kart-io-network; do
        if docker network ls --format '{{.Name}}' | grep -q "^${network}$"; then
            log_debug "删除网络: $network"
            docker network rm $network 2>/dev/null || true
        fi
    done
    
    log_info "✅ 旧资源清理完成"
}

# 拉取最新镜像
pull_images() {
    log_info "拉取最新Docker镜像..."
    
    images=(
        "otel/opentelemetry-collector-contrib:0.132.0"
        "victoriametrics/victoria-logs:v1.28.0"
        "jaegertracing/all-in-one:1.57"
        "prom/prometheus:v2.51.0"
    )
    
    for image in "${images[@]}"; do
        log_debug "拉取镜像: $image"
        docker pull "$image"
    done
    
    log_info "✅ 镜像拉取完成"
}

# 部署OTLP栈
deploy_stack() {
    log_info "部署OTLP技术栈..."
    
    # 确保在正确的目录
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cd "$SCRIPT_DIR"
    
    # 检查配置文件
    required_files=(
        "otel-agent-config.yaml"
        "otel-collector-config.yaml"
        "start-network.sh"
        "start-victorialogs.sh"
        "start-collector.sh"
        "start-agent.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log_error "缺少文件: $file"
            exit 1
        fi
    done
    
    # 按顺序启动服务
    log_info "创建Docker网络..."
    ./start-network.sh
    
    log_info "启动VictoriaLogs..."
    ./start-victorialogs.sh
    
    log_info "启动OTEL Collector..."
    ./start-collector.sh
    
    log_info "启动OTEL Agent..."
    ./start-agent.sh
    
    log_info "✅ OTLP技术栈部署完成"
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务启动完成..."
    
    services=(
        "VictoriaLogs:http://localhost:9428/health"
        "OTEL Agent:http://localhost:13133/"
        "OTEL Collector:http://localhost:13134/"
    )
    
    for service_info in "${services[@]}"; do
        service_name=$(echo "$service_info" | cut -d: -f1)
        service_url=$(echo "$service_info" | cut -d: -f2-)
        
        log_debug "检查服务: $service_name"
        
        # 等待服务启动（最多等待60秒）
        for i in {1..12}; do
            if curl -s --max-time 5 "$service_url" > /dev/null 2>&1; then
                log_info "✅ $service_name 已就绪"
                break
            elif [[ $i -eq 12 ]]; then
                log_warn "⚠️  $service_name 可能未正确启动"
            else
                log_debug "$service_name 启动中... ($i/12)"
                sleep 5
            fi
        done
    done
    
    log_info "✅ 服务启动检查完成"
}

# 显示服务状态
show_status() {
    log_info "OTLP技术栈服务状态:"
    echo
    docker ps --filter "name=kart-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo
    
    log_info "服务访问地址:"
    echo -e "${BLUE}OTEL Agent (gRPC):${NC}     localhost:4327"
    echo -e "${BLUE}OTEL Agent (HTTP):${NC}     localhost:4328"
    echo -e "${BLUE}OTEL Collector (gRPC):${NC} localhost:4317"
    echo -e "${BLUE}OTEL Collector (HTTP):${NC} localhost:4318"
    echo -e "${BLUE}VictoriaLogs:${NC}          http://localhost:9428"
    echo
    
    log_info "健康检查地址:"
    echo -e "${BLUE}Agent Health:${NC}          http://localhost:13133/"
    echo -e "${BLUE}Collector Health:${NC}      http://localhost:13134/"
    echo -e "${BLUE}VictoriaLogs Health:${NC}   http://localhost:9428/health"
    echo
    
    log_info "指标地址:"
    echo -e "${BLUE}Agent Metrics:${NC}         http://localhost:8888/metrics"
    echo -e "${BLUE}Collector Metrics:${NC}     http://localhost:8889/metrics"
    echo -e "${BLUE}VictoriaLogs Metrics:${NC}  http://localhost:9428/metrics"
    echo
}

# 显示使用说明
show_usage() {
    echo "OTLP Stack 部署脚本"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -c, --clean    仅清理旧资源（不部署）"
    echo "  --no-pull      跳过镜像拉取"
    echo "  --skip-wait    跳过服务就绪检查"
    echo
    echo "示例:"
    echo "  $0                # 完整部署"
    echo "  $0 --clean        # 仅清理"
    echo "  $0 --no-pull      # 部署但不拉取新镜像"
    echo
}

# 主函数
main() {
    local clean_only=false
    local skip_pull=false
    local skip_wait=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -c|--clean)
                clean_only=true
                shift
                ;;
            --no-pull)
                skip_pull=true
                shift
                ;;
            --skip-wait)
                skip_wait=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    echo "🚀 OTLP Stack 部署脚本启动"
    echo "================================================"
    
    # 执行部署步骤
    check_dependencies
    cleanup_old
    
    if [[ "$clean_only" == true ]]; then
        log_info "🧹 仅执行清理操作完成"
        exit 0
    fi
    
    if [[ "$skip_pull" != true ]]; then
        pull_images
    fi
    
    deploy_stack
    
    if [[ "$skip_wait" != true ]]; then
        wait_for_services
    fi
    
    show_status
    
    echo
    echo "================================================"
    log_info "🎉 OTLP技术栈部署完成！"
    echo
    log_info "流程: 应用程序 → OTEL Agent(4327) → OTEL Collector(4317) → VictoriaLogs(9428)"
    echo
    log_info "下一步："
    echo "1. 运行测试脚本: ./test.sh"
    echo "2. 查看服务日志: docker logs kart-otel-agent"
    echo "3. 停止服务: ./stop.sh"
}

# 执行主函数
main "$@"