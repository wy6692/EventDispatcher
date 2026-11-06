# 运维排障与高可用指南

本文档介绍 EventDispatcher 的日常运维、监控指标、故障排障以及数据备份策略。

---

## 1. 监控与健康检查

### 存活与健康探针 (Liveness / Readiness)
网关内置 `/healthz` 端点，支持 Kubernetes 与各类负载均衡器探活：

```bash
curl -i http://127.0.0.1:8080/healthz
```

**响应规范**：
- **正常状态 (HTTP 200)**：
  ```json
  {
    "code": 0,
    "message": "success",
    "data": { "status": "ok" },
    "traceId": "trace_..."
  }
  ```
- **异常状态 (HTTP 503)**：
  当底层 SQLite 数据库不可读写时返回，提示服务处于降级状态。

---

## 2. 投递重试与指数退避机制

下游业务系统偶发网络抖动或发布重启时，EventDispatcher 的后台调度 Worker 会自动执行可靠投递：

### 退避时间表 (Default 6 Attempts)
| 轮次 | 退避等待间隔 | 说明 |
| :--- | :--- | :--- |
| 第 1 次失败 | 约 1 秒 | 首次偶发抖动快速重试 |
| 第 2 次失败 | 约 5 秒 | 短暂故障恢复 |
| 第 3 次失败 | 约 30 秒 | 进程发布阶段 |
| 第 4 次失败 | 约 2 分钟 | 服务短暂中断 |
| 第 5 次失败 | 约 10 分钟 | 较长时间中断 |
| 第 6 次失败 | 约 30 分钟 | 最终重试轮次 |

若超过最大重试次数（默认 6 轮）仍然失败，任务状态将置为 `failed`（终态），不再自动调度，等待人工介入。

### 手动重放失败任务
可通过 Web 控制台（**投递调度与重试** 页面）点击「重放任务」，或调用 API：
```bash
curl -X POST http://127.0.0.1:8080/admin/deliveries/{deliveryId}/retry \
  -H "Authorization: Bearer your-admin-token"
```
执行后状态将重置为 `pending`，Worker 会在下一调度周期立即执行投递。

---

## 3. 存储引擎维护与热备份

EventDispatcher 采用纯 Go 实现的嵌入式 SQLite（无需 CGO），数据完整保存在单文件中（默认 `data/dispatcher.db`）。

### 性能调优特性
- 自动开启 **WAL (Write-Ahead Logging)** 模式，读写完全并发不阻塞。
- 连接池采用“单写多读”隔离架构，杜绝并发写入时的 `database is locked` 异常。

### 热备份方案
由于开启了 WAL 模式，备份时可直接使用 SQLite 的在线备份接口，或在系统闲时直接复制 `.db` 及其关联的 `.db-wal` 文件：
```bash
# 复制备份
cp data/dispatcher.db /backup/dispatcher_$(date +%Y%m%d).db
```

---

## 4. 常见排错场景

### Q1: 微信后台提示“Token 验证失败”
1. 检查服务是否已绑定对应域名的 80/443 端口，外部网络能否直接访问该 URL。
2. 检查 EventDispatcher 中填写的 `Token` 与微信公众平台填写的 `Token` 是否完全一致（注意空格与特殊符号）。
3. 检查反向代理（如 Nginx）是否正确透传了 `GET` 参数（包括 `signature`, `timestamp`, `nonce`, `echostr`）。

### Q2: 微信后台提示“应用响应超时”
EventDispatcher 收到请求后采用**毫秒级回复 `success`** 设计。如果仍然超时，请排查：
1. 外部 HTTPS 证书握手延迟或网络链路丢包。
2. 服务器磁盘 I/O 达到极限，导致 SQLite 写入挂起。
