# 日志轮转集成示例

这个示例展示了如何将 kart-io/logger 与 lumberjack 日志轮转库集成，实现生产环境级别的日志管理。

## 功能特性

- ✅ 基于文件大小的自动轮转
- ✅ 基于时间的日志清理
- ✅ 日志压缩节省磁盘空间
- ✅ 多输出支持（控制台+文件）
- ✅ 生产环境配置示例
- ✅ 完整的 Logger 接口实现

## 运行示例

### 1. 安装依赖

```bash
cd example/rotation
go mod tidy
```

### 2. 运行示例

```bash
go run main.go
```

### 3. 查看生成的日志文件

```bash
ls -la logs/
```

你将看到以下文件：
- `basic.log*` - 基本轮转示例的日志文件
- `advanced.log*` - 高级配置的日志文件  
- `multi.log*` - 多输出示例的日志文件
- `production.log*` - 生产环境配置的日志文件

## 示例说明

### 1. 基本轮转
- 文件大小：1MB 触发轮转
- 备份数量：保留 3 个备份文件
- 保留时间：7 天后自动删除
- 压缩：关闭（便于查看）

### 2. 高级配置
- 文件大小：2MB 触发轮转
- 备份数量：保留 5 个备份文件
- 保留时间：3 天后自动删除
- 压缩：开启（节省空间）

### 3. 多输出
- 同时输出到控制台和文件
- 控制台使用 console 格式
- 文件使用 JSON 格式

### 4. 生产环境
- 文件大小：100MB（生产级别）
- 备份数量：15 个（约 15 天）
- 保留时间：15 天
- 压缩：开启
- 结构化字段：包含服务信息

## 配置参数说明

### Lumberjack 配置
```go
&lumberjack.Logger{
    Filename:   "./logs/app.log",  // 日志文件路径
    MaxSize:    100,               // 单文件最大尺寸（MB）
    MaxBackups: 15,                // 保留的备份文件数量
    MaxAge:     15,                // 文件保留天数
    Compress:   true,              // 是否压缩旧文件
    LocalTime:  true,              // 使用本地时间命名
}
```

### 轮转策略选择建议

| 应用类型 | MaxSize | MaxBackups | MaxAge | Compress | 说明 |
|---------|---------|------------|--------|----------|------|
| 开发环境 | 10MB | 3 | 1天 | false | 快速轮转，便于调试 |
| 测试环境 | 50MB | 7 | 3天 | false | 中等保留，不压缩便于查看 |
| 生产环境 | 200MB | 15 | 15天 | true | 长期保留，压缩节省空间 |
| 高并发 | 100MB | 24 | 7天 | true | 频繁轮转，短期保留 |

## 集成到你的项目

### 1. 复制 RotationLogger 实现

将 `RotationLogger` 结构体和相关方法复制到你的项目中。

### 2. 根据需要调整配置

```go
// 根据你的需求调整轮转参数
rotateWriter := &lumberjack.Logger{
    Filename:   "/var/log/myapp/app.log",
    MaxSize:    200,  // 根据日志量调整
    MaxBackups: 15,   // 根据保留需求调整
    MaxAge:     15,   // 根据合规要求调整
    Compress:   true, // 生产环境建议开启
    LocalTime:  true,
}

logger := NewRotationLogger(rotateWriter, core.InfoLevel, "json")
```

### 3. 添加监控（可选）

```go
// 监控日志轮转事件
type MonitoredRotation struct {
    *lumberjack.Logger
    rotationCount int64
}

func (m *MonitoredRotation) Write(p []byte) (n int, err error) {
    oldSize := m.size()
    n, err = m.Logger.Write(p)
    if m.size() < oldSize {
        atomic.AddInt64(&m.rotationCount, 1)
        // 发送轮转事件到监控系统
    }
    return n, err
}
```

## 注意事项

1. **文件权限**：确保应用有写入日志目录的权限
2. **磁盘空间**：监控磁盘使用，设置合理的保留策略
3. **并发安全**：lumberjack 是并发安全的
4. **信号处理**：生产环境建议添加 USR1 信号处理用于手动轮转
5. **性能影响**：频繁的小文件轮转可能影响性能

## 扩展功能

- [ ] 添加基于时间的轮转（每日/每小时）
- [ ] 集成 Prometheus 监控指标
- [ ] 支持远程日志传输
- [ ] 添加日志文件完整性校验
- [ ] 实现自定义轮转触发器

## 生产环境部署建议

1. **使用绝对路径**：避免相对路径导致的问题
2. **设置适当的文件权限**：`0640` 或 `0644`
3. **配置日志目录的定期清理**：除了 MaxAge 之外的额外保障
4. **监控磁盘使用情况**：设置告警阈值
5. **备份重要日志**：对于审计日志考虑异地备份