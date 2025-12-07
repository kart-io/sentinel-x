#!/bin/bash

# 生产环境日志轮转部署脚本
# 用法: ./deploy.sh [app_name] [log_dir] [user]

set -e

APP_NAME=${1:-myapp}
LOG_DIR=${2:-/var/log/$APP_NAME}
APP_USER=${3:-$APP_NAME}

echo "=== 日志轮转生产环境部署脚本 ==="
echo "应用名称: $APP_NAME"
echo "日志目录: $LOG_DIR"
echo "运行用户: $APP_USER"
echo

# 检查是否为 root 用户
if [[ $EUID -ne 0 ]]; then
   echo "此脚本需要 root 权限执行"
   exit 1
fi

# 1. 创建应用用户（如果不存在）
echo "1. 检查应用用户..."
if ! id "$APP_USER" &>/dev/null; then
    echo "创建用户: $APP_USER"
    useradd -r -s /bin/false "$APP_USER"
else
    echo "用户 $APP_USER 已存在"
fi

# 2. 创建日志目录
echo "2. 创建日志目录..."
mkdir -p "$LOG_DIR"
chown "$APP_USER:$APP_USER" "$LOG_DIR"
chmod 755 "$LOG_DIR"
echo "日志目录创建完成: $LOG_DIR"

# 3. 设置 logrotate 配置
echo "3. 配置 logrotate..."
cat > "/etc/logrotate.d/$APP_NAME" << EOF
# $APP_NAME 应用日志轮转配置
# 生成时间: $(date)

$LOG_DIR/*.log {
    # 每天轮转
    daily
    
    # 保留 15 天
    rotate 15
    
    # 文件不存在时不报错
    missingok
    
    # 空文件不轮转
    notifempty
    
    # 压缩旧文件
    compress
    
    # 延迟压缩
    delaycompress
    
    # 新文件权限
    create 0644 $APP_USER $APP_USER
    
    # 文件大小限制（紧急轮转）
    size 200M
    
    # 轮转后处理
    postrotate
        # 发送 USR1 信号重新打开日志文件
        if [ -f /var/run/$APP_NAME.pid ]; then
            /bin/kill -USR1 \$(cat /var/run/$APP_NAME.pid) 2>/dev/null || true
        fi
        
        # 或者使用 systemctl 重载（如果使用 systemd）
        # /bin/systemctl reload $APP_NAME 2>/dev/null || true
    endscript
}

# 错误日志单独配置（保留更长时间）
$LOG_DIR/error.log {
    daily
    rotate 30
    missingok
    notifempty
    compress
    delaycompress
    create 0644 $APP_USER $APP_USER
    size 100M
    
    postrotate
        if [ -f /var/run/$APP_NAME.pid ]; then
            /bin/kill -USR1 \$(cat /var/run/$APP_NAME.pid) 2>/dev/null || true
        fi
    endscript
}

# 访问日志配置（保留较短时间）
$LOG_DIR/access.log {
    daily
    rotate 7
    missingok
    notifempty
    compress
    delaycompress
    create 0644 $APP_USER $APP_USER
    size 500M
    
    postrotate
        if [ -f /var/run/$APP_NAME.pid ]; then
            /bin/kill -USR1 \$(cat /var/run/$APP_NAME.pid) 2>/dev/null || true
        fi
    endscript
}
EOF

echo "logrotate 配置已创建: /etc/logrotate.d/$APP_NAME"

# 4. 测试 logrotate 配置
echo "4. 测试 logrotate 配置..."
if logrotate -d "/etc/logrotate.d/$APP_NAME"; then
    echo "logrotate 配置语法检查通过"
else
    echo "logrotate 配置语法错误，请检查"
    exit 1
fi

# 5. 创建示例 systemd 服务文件（可选）
echo "5. 创建示例 systemd 服务文件..."
cat > "/etc/systemd/system/$APP_NAME.service.example" << EOF
[Unit]
Description=$APP_NAME Application
After=network.target

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
WorkingDirectory=/opt/$APP_NAME
ExecStart=/opt/$APP_NAME/bin/$APP_NAME
Restart=always
RestartSec=5

# 环境变量
Environment=LOG_LEVEL=INFO
Environment=LOG_FORMAT=json
Environment=LOG_OUTPUT_PATHS=$LOG_DIR/app.log

# PID 文件
PIDFile=/var/run/$APP_NAME.pid

# 日志轮转支持
ExecReload=/bin/kill -USR1 \$MAINPID

# 安全设置
NoNewPrivileges=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
EOF

echo "systemd 服务示例文件已创建: /etc/systemd/system/$APP_NAME.service.example"

# 6. 创建管理脚本
echo "6. 创建管理脚本..."
cat > "/usr/local/bin/$APP_NAME-log-rotate" << EOF
#!/bin/bash
# $APP_NAME 日志轮转管理脚本

case "\$1" in
    rotate)
        echo "手动触发日志轮转..."
        logrotate -f "/etc/logrotate.d/$APP_NAME"
        ;;
    status)
        echo "日志文件状态:"
        ls -lah "$LOG_DIR/"
        echo
        echo "轮转历史:"
        if [ -f "/var/lib/logrotate/logrotate.state" ]; then
            grep "$APP_NAME" "/var/lib/logrotate/logrotate.state" || echo "未找到轮转记录"
        fi
        ;;
    test)
        echo "测试 logrotate 配置..."
        logrotate -d "/etc/logrotate.d/$APP_NAME"
        ;;
    reload)
        echo "重载应用日志文件..."
        if [ -f "/var/run/$APP_NAME.pid" ]; then
            kill -USR1 \$(cat "/var/run/$APP_NAME.pid")
            echo "已发送 USR1 信号"
        else
            echo "PID 文件不存在: /var/run/$APP_NAME.pid"
        fi
        ;;
    *)
        echo "用法: \$0 {rotate|status|test|reload}"
        echo "  rotate  - 手动触发日志轮转"
        echo "  status  - 查看日志文件状态"
        echo "  test    - 测试轮转配置"
        echo "  reload  - 重载应用日志文件"
        exit 1
        ;;
esac
EOF

chmod +x "/usr/local/bin/$APP_NAME-log-rotate"
echo "管理脚本已创建: /usr/local/bin/$APP_NAME-log-rotate"

# 7. 创建监控脚本
echo "7. 创建监控脚本..."
cat > "/usr/local/bin/$APP_NAME-log-monitor" << EOF
#!/bin/bash
# $APP_NAME 日志监控脚本

LOG_DIR="$LOG_DIR"
ALERT_SIZE_MB=1000  # 单个文件超过 1GB 时告警
ALERT_TOTAL_MB=5000 # 总大小超过 5GB 时告警

# 检查单个文件大小
check_file_sizes() {
    echo "检查单个文件大小..."
    find "\$LOG_DIR" -name "*.log" -size +\${ALERT_SIZE_MB}M -exec ls -lh {} \; | while read line; do
        echo "WARNING: 发现大文件: \$line"
    done
}

# 检查总磁盘使用
check_total_usage() {
    echo "检查总磁盘使用..."
    total_kb=\$(du -sk "\$LOG_DIR" | cut -f1)
    total_mb=\$((total_kb / 1024))
    
    echo "日志目录总大小: \${total_mb}MB"
    
    if [ \$total_mb -gt \$ALERT_TOTAL_MB ]; then
        echo "ERROR: 日志目录总大小超出阈值 (\${total_mb}MB > \${ALERT_TOTAL_MB}MB)"
    fi
}

# 检查轮转状态
check_rotation_status() {
    echo "检查轮转状态..."
    if [ -f "/var/lib/logrotate/logrotate.state" ]; then
        echo "最近轮转记录:"
        grep "$APP_NAME" "/var/lib/logrotate/logrotate.state" | tail -5
    fi
}

# 检查磁盘空间
check_disk_space() {
    echo "检查磁盘空间..."
    df -h "\$LOG_DIR"
}

echo "=== $APP_NAME 日志监控报告 - \$(date) ==="
echo
check_file_sizes
echo
check_total_usage
echo
check_rotation_status
echo
check_disk_space
EOF

chmod +x "/usr/local/bin/$APP_NAME-log-monitor"
echo "监控脚本已创建: /usr/local/bin/$APP_NAME-log-monitor"

# 8. 设置 crontab（可选）
echo "8. 设置定时监控..."
if ! crontab -l 2>/dev/null | grep -q "$APP_NAME-log-monitor"; then
    (crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/$APP_NAME-log-monitor >> /var/log/$APP_NAME-monitor.log 2>&1") | crontab -
    echo "已添加定时监控任务（每天凌晨2点执行）"
else
    echo "定时监控任务已存在"
fi

# 9. 总结
echo
echo "=== 部署完成 ==="
echo "配置文件："
echo "  logrotate: /etc/logrotate.d/$APP_NAME"
echo "  systemd示例: /etc/systemd/system/$APP_NAME.service.example"
echo
echo "管理脚本："
echo "  日志轮转: /usr/local/bin/$APP_NAME-log-rotate"
echo "  日志监控: /usr/local/bin/$APP_NAME-log-monitor"
echo
echo "日志目录: $LOG_DIR"
echo "运行用户: $APP_USER"
echo
echo "下一步操作："
echo "1. 根据需要调整 /etc/logrotate.d/$APP_NAME 中的配置"
echo "2. 复制并修改 systemd 服务文件"
echo "3. 在应用中实现 USR1 信号处理"
echo "4. 测试轮转: $APP_NAME-log-rotate test"
echo "5. 手动轮转: $APP_NAME-log-rotate rotate"
echo "6. 查看状态: $APP_NAME-log-rotate status"
echo "7. 监控日志: $APP_NAME-log-monitor"