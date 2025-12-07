#!/bin/bash

# OTLP Stack 测试脚本
# 用途：测试完整的 Application → Agent → Collector → VictoriaLogs 链路
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

# 检查服务状态
check_services() {
    log_info "检查OTLP服务状态..."
    
    services=(
        "OTEL Agent (Health)|http://localhost:13133/|Agent健康检查"
        "OTEL Collector (Health)|http://localhost:13134/|Collector健康检查"  
        "VictoriaLogs (Health)|http://localhost:9428/health|VictoriaLogs健康检查"
    )
    
    local all_ok=true
    
    for service_info in "${services[@]}"; do
        service_name=$(echo "$service_info" | cut -d'|' -f1)
        service_url=$(echo "$service_info" | cut -d'|' -f2)
        service_desc=$(echo "$service_info" | cut -d'|' -f3)
        
        if curl -s --max-time 5 "$service_url" > /dev/null 2>&1; then
            log_info "✅ $service_name - $service_desc"
        else
            log_error "❌ $service_name - $service_desc"
            all_ok=false
        fi
    done
    
    if [[ "$all_ok" != true ]]; then
        log_error "部分服务未就绪，请检查部署状态"
        exit 1
    fi
    
    log_info "✅ 所有服务状态正常"
}

# 测试端口连通性
test_connectivity() {
    log_info "测试OTLP端口连通性..."
    
    ports=(
        "4327:OTEL Agent gRPC"
        "4328:OTEL Agent HTTP"
        "4317:OTEL Collector gRPC"
        "4318:OTEL Collector HTTP"
        "9428:VictoriaLogs HTTP"
    )
    
    for port_info in "${ports[@]}"; do
        port=$(echo "$port_info" | cut -d: -f1)
        desc=$(echo "$port_info" | cut -d: -f2)
        
        if timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
            log_info "✅ 端口 $port ($desc) 连通"
        else
            log_error "❌ 端口 $port ($desc) 无法连接"
        fi
    done
}

# 发送测试日志到Agent
send_test_logs() {
    log_info "发送测试日志到OTEL Agent..."
    
    # 切换到Go项目目录
    PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    cd "$PROJECT_DIR"
    
    # 检查Go项目是否存在
    if [[ ! -f "go.mod" ]]; then
        log_error "未找到Go项目，请确保在正确的目录执行脚本"
        exit 1
    fi
    
    # 创建测试程序
    cat > /tmp/chain_test_main.go << 'EOF'
package main

import (
	"fmt"
	"time"
	
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== OTLP链路测试 ===")
	fmt.Println("流程: 应用程序 → Agent(4327) → Collector(4317) → VictoriaLogs(9428)")
	
	testID := fmt.Sprintf("chain_test_%d", time.Now().Unix())
	
	// 测试Agent gRPC
	testAgentGRPC(testID)
	
	// 测试Agent HTTP  
	testAgentHTTP(testID)
	
	fmt.Printf("📤 测试完成，test_id: %s\n", testID)
	fmt.Println("等待5秒让日志传输...")
	time.Sleep(5 * time.Second)
}

func testAgentGRPC(testID string) {
	fmt.Println("\n1. 测试Agent gRPC链路")
	
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327",  // Agent gRPC
			Protocol: "grpc",
			Timeout:  5 * time.Second,
		},
	}
	
	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("❌ Agent gRPC Logger创建失败: %v\n", err)
		return
	}
	
	logger.Infow("OTLP链路测试 - gRPC",
		"test_id", testID,
		"component", "agent",
		"protocol", "grpc",
		"port", 4327,
		"chain", "app->agent->collector->victorialogs",
	)
	
	fmt.Println("✅ Agent gRPC测试完成")
}

func testAgentHTTP(testID string) {
	fmt.Println("\n2. 测试Agent HTTP链路")
	
	opt := &option.LogOption{
		Engine:      "slog",
		Level:       "INFO", 
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4328",  // Agent HTTP
			Protocol: "http",
			Timeout:  5 * time.Second,
		},
	}
	
	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("❌ Agent HTTP Logger创建失败: %v\n", err)
		return
	}
	
	logger.Infow("OTLP链路测试 - HTTP",
		"test_id", testID,
		"component", "agent", 
		"protocol", "http",
		"port", 4328,
		"chain", "app->agent->collector->victorialogs",
	)
	
	fmt.Println("✅ Agent HTTP测试完成")
}
EOF
    
    # 运行测试程序
    log_info "执行Go测试程序..."
    if go run /tmp/chain_test_main.go; then
        log_info "✅ 测试日志发送成功"
    else
        log_error "❌ 测试日志发送失败"
        return 1
    fi
    
    # 清理临时文件
    rm -f /tmp/chain_test_main.go
}

# 验证日志是否到达VictoriaLogs
verify_logs() {
    log_info "验证日志是否到达VictoriaLogs..."
    
    # 等待日志处理
    sleep 10
    
    # 查询最近的日志
    local response
    if response=$(curl -s "http://localhost:9428/select/logsql/query?query=*&limit=5" 2>/dev/null); then
        if [[ -n "$response" ]]; then
            local log_count=$(echo "$response" | wc -l)
            if [[ "$log_count" -gt 0 ]]; then
                log_info "✅ VictoriaLogs中找到 $log_count 条日志记录"
                
                # 显示最新的日志
                echo "$response" | head -1 | jq -r '"最新日志: " + ._msg + " (时间: " + ._time + ")"' 2>/dev/null || true
                
                # 查询测试日志
                local test_response
                if test_response=$(curl -s "http://localhost:9428/select/logsql/query?query=test_id:chain_test*&limit=3" 2>/dev/null); then
                    local test_count=$(echo "$test_response" | wc -l)
                    if [[ "$test_count" -gt 0 ]]; then
                        log_info "🎉 找到 $test_count 条测试链路日志"
                        echo "$test_response" | jq -r '"- " + ._msg + " (" + .protocol + ")"' 2>/dev/null || true
                    else
                        log_warn "⚠️  未找到测试链路日志，可能链路存在问题"
                    fi
                fi
            else
                log_warn "⚠️  VictoriaLogs中未找到日志记录"
            fi
        else
            log_warn "⚠️  VictoriaLogs返回空响应"
        fi
    else
        log_error "❌ 无法查询VictoriaLogs"
        return 1
    fi
}

# 显示链路监控信息
show_monitoring() {
    log_info "OTLP链路监控信息:"
    echo
    
    # VictoriaLogs统计
    if stats=$(curl -s "http://localhost:9428/metrics" 2>/dev/null | grep "vl_http_requests_total.*opentelemetry" || true); then
        if [[ -n "$stats" ]]; then
            echo -e "${BLUE}VictoriaLogs OTLP请求统计:${NC}"
            echo "$stats"
        fi
    fi
    
    echo
    log_info "监控面板地址:"
    echo -e "${BLUE}VictoriaLogs查询:${NC}    http://localhost:9428/select/logsql/query?query=*"
    echo -e "${BLUE}Agent指标:${NC}          http://localhost:8888/metrics"
    echo -e "${BLUE}Collector指标:${NC}      http://localhost:8889/metrics"
}

# 性能测试
performance_test() {
    log_info "执行性能测试（可选）..."
    
    read -p "是否执行性能测试？这将发送1000条测试日志 (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "跳过性能测试"
        return 0
    fi
    
    # 切换到Go项目目录
    PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    cd "$PROJECT_DIR"
    
    # 创建性能测试程序
    cat > /tmp/perf_test.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"time"
	
	"github.com/kart-io/logger"
	"github.com/kart-io/logger/option"
)

func main() {
	fmt.Println("=== OTLP性能测试 ===")
	
	opt := &option.LogOption{
		Engine:      "zap",
		Level:       "INFO",
		Format:      "json",
		OutputPaths: []string{"stdout"},
		OTLP: &option.OTLPOption{
			Endpoint: "127.0.0.1:4327",
			Protocol: "grpc", 
			Timeout:  5 * time.Second,
		},
	}
	
	logger, err := logger.New(opt)
	if err != nil {
		fmt.Printf("Logger创建失败: %v\n", err)
		return
	}
	
	const numLogs = 1000
	const numWorkers = 10
	
	start := time.Now()
	
	var wg sync.WaitGroup
	logsChan := make(chan int, numLogs)
	
	// 启动工作协程
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for logID := range logsChan {
				logger.Infow("性能测试日志",
					"log_id", logID,
					"worker_id", workerID,
					"timestamp", time.Now(),
					"test_type", "performance",
				)
			}
		}(i)
	}
	
	// 发送日志任务
	for i := 0; i < numLogs; i++ {
		logsChan <- i
	}
	close(logsChan)
	
	// 等待完成
	wg.Wait()
	
	duration := time.Since(start)
	fmt.Printf("性能测试完成: %d条日志, 耗时: %v, 平均: %.2f日志/秒\n", 
		numLogs, duration, float64(numLogs)/duration.Seconds())
}
EOF
    
    log_info "启动性能测试..."
    if go run /tmp/perf_test.go; then
        log_info "✅ 性能测试完成"
    else
        log_error "❌ 性能测试失败"
    fi
    
    # 清理临时文件
    rm -f /tmp/perf_test.go
}

# 显示使用说明
show_usage() {
    echo "OTLP Stack 测试脚本"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help      显示此帮助信息"
    echo "  --check-only    仅检查服务状态"
    echo "  --perf          执行性能测试"
    echo "  --no-verify     跳过日志验证"
    echo
    echo "示例:"
    echo "  $0              # 完整测试"
    echo "  $0 --check-only # 仅检查状态"
    echo "  $0 --perf       # 包含性能测试"
    echo
}

# 主函数
main() {
    local check_only=false
    local run_perf=false
    local skip_verify=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            --check-only)
                check_only=true
                shift
                ;;
            --perf)
                run_perf=true
                shift
                ;;
            --no-verify)
                skip_verify=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    echo "🧪 OTLP Stack 测试脚本启动"
    echo "================================================"
    
    # 检查必要工具
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装，无法执行测试"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warn "jq 未安装，日志解析功能受限"
    fi
    
    # 执行测试步骤
    check_services
    test_connectivity
    
    if [[ "$check_only" == true ]]; then
        log_info "🔍 仅执行状态检查完成"
        exit 0
    fi
    
    send_test_logs
    
    if [[ "$skip_verify" != true ]]; then
        verify_logs
    fi
    
    show_monitoring
    
    if [[ "$run_perf" == true ]]; then
        performance_test
    fi
    
    echo
    echo "================================================"
    log_info "🎉 OTLP链路测试完成！"
    echo
    log_info "测试流程: 应用程序 → Agent(4327) → Collector(4317) → VictoriaLogs(9428)"
    echo
    log_info "如果测试失败，请检查："
    echo "1. 所有服务是否正常运行: docker ps --filter 'name=kart-'"
    echo "2. 服务日志: docker logs kart-otel-agent"
    echo "3. 网络连通性: ./test.sh --check-only"
}

# 执行主函数
main "$@"