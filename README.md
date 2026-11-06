<div align="center">

  <img src="assets/logo.svg" alt="EventDispatcher Logo" width="96" height="96">

  <h1>EventDispatcher</h1>

  <p><strong>企业级微信公众号事件与消息统一分发网关</strong></p>
  <p>突破微信单一回调限制，解耦多业务系统消费，内置规则过滤、可靠投递与管理控制台。</p>

  <p align="center">
    <img src="assets/badges/go-version.svg" alt="Go Version">
    <img src="assets/badges/build-status.svg" alt="Build Status">
    <img src="assets/badges/code-quality.svg" alt="Code Quality">
    <img src="assets/badges/license.svg" alt="License">
    <img src="assets/badges/docker.svg" alt="Docker Ready">
    <img src="assets/badges/storage.svg" alt="SQLite Pure Go">
  </p>

  <p align="center">
    <a href="#快速上手">快速上手</a> &bull;
    <a href="#开箱即用的-web-控制台">控制台展示</a> &bull;
    <a href="#核心特性">核心特性</a> &bull;
    <a href="#架构速览">系统架构</a> &bull;
    <a href="#文档中心">完整文档</a>
  </p>

</div>

---

## 控制台概览

EventDispatcher 内置开箱即用的现代化暗黑管理控制台，由 Go 单二进制原生嵌入，**零外部前端依赖**：

<p align="center">
  <img src="assets/dashboard-preview.svg" alt="EventDispatcher Web Console Showcase" width="100%">
</p>

---

## 解决的业务痛点

微信公众平台仅允许配置 **一个服务器地址 (URL)**，但在企业实际落地中，多个系统往往需要消费微信事件：
- 自定义菜单系统（响应特定 CLICK 事件）；
- 渠道与用户增长中心（识别带参二维码关注归因）；
- 智能客服与工单系统（消费普通文本、图片与语音消息）；
- 安全合规与全量审计系统（归档所有交互日志）。

| 痛点场景 | 传统单体转发方案 | EventDispatcher 网关方案 |
| :--- | :--- | :--- |
| **系统耦合度** | 下游系统强依赖单一转发层，变更风险大 | 业务系统独立解耦，按需注册 Webhook 订阅 |
| **微信响应时效** | 链路多级同步透传，极易触发 5 秒超时 | 收到推送后立即落库并回复 `success`（< 15ms） |
| **凭据安全** | 微信 Token/AES 密钥散落在各业务方 | 微信凭据仅由网关托管，下游出站采用 HMAC 验签 |
| **可靠性保证** | 业务端自行实现重试与去重，极易丢失事件 | 内置全局幂等去重与指数退避重试调度引擎 |

---

## 核心特性

- **协议兼容完备**：完整支持明文模式、兼容模式及安全模式（AES-256-CBC），自动完成 PKCS#7 逆填充与解析。
- **高并发极速响应**：毫秒级持久化并向微信输出 `success`，业务分发由后台 Worker 异步调度，彻底杜绝超时重推。
- **声明式规则路由**：各订阅方支持通过 JSON Filter 精准过滤（`msgTypes`、`events`、`eventKeyPrefixes`）。
- **可靠的阶梯重试**：下游返回非 2xx 或请求超时时，自动按阶梯间隔退避重试（最多 6 轮），支持一键手动重放。
- **零外部运维依赖**：基于纯 Go 实现的嵌入式 SQLite（开启 WAL 模式），单二进制文件即可运行，无需 MySQL 或 Redis。
- **工业级安全设计**：管理接口采用恒定时间 Token 校验，出站消息支持 HMAC-SHA256 签名，接口敏感字段零明文泄露。

---

## 快速上手

### 方式一：源码编译与本地启动

```bash
# 1. 克隆代码仓库
git clone https://github.com/wy6692/EventDispatcher.git
cd EventDispatcher

# 2. 设置管理员口令并启动
export ADMIN_TOKEN="your-secure-admin-token"
export ADDR=":8080"
export DB_PATH="data/dispatcher.db"

go run ./cmd/dispatcher
```

### 方式二：Docker Compose 一键部署

```bash
docker compose up -d
```

启动完成后，在浏览器中打开控制台即可开始配置：
```text
http://localhost:8080/ui/
```

---

## 架构速览

<p align="center">
  <img src="assets/architecture.svg" alt="EventDispatcher Architecture" width="100%">
</p>

- **接入层**：验证微信服务器合法性，毫秒级解密验签与快速应答；
- **领域与编排层**：计算全局幂等指纹，通过规则过滤引擎精准生成投递任务；
- **持久化层**：SQLite WAL 单写多读隔离，保证高并发读写零锁冲突；
- **调度层**：抢占式后台 Worker 执行签名投递与指数退避重试。

---

## 文档中心

详细的技术文档与使用指南请查阅 `docs/` 目录：

- **[系统架构与流程时序 (docs/architecture.md)](docs/architecture.md)**：分层模型、WAL 并发控制与事件完整生命周期。
- **[微信公众平台接入指南 (docs/wechat-setup.md)](docs/wechat-setup.md)**：微信开发者配置、Token 验证、多服务号隔离。
- **[规则路由与过滤引擎 (docs/routing-rules.md)](docs/routing-rules.md)**：声明式 JSON Filter 语法与典型业务配置范式。
- **[RESTful API 与验签规范 (docs/api.md)](docs/api.md)**：管理接口定义、出站 Webhook 载荷结构及多语言验签示例。
- **[运维监控与排障手册 (docs/ops-guide.md)](docs/ops-guide.md)**：探活探针、SQLite 热备份、任务重试与排错指南。
- **[开发与质量工程规范 (AGENT.md)](AGENT.md)**：Clean Architecture 分层约束、Git 提交与测试标准。

---

## 许可证

本项目采用 [MIT 许可证](LICENSE) 开源。
