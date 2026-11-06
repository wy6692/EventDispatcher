<div align="center">

# EventDispatcher

**企业级微信公众号事件/消息统一网关与异步分发中间件**

<p align="center">
  <img src="assets/badges/go-version.svg" alt="Go Version">
  <img src="assets/badges/build-status.svg" alt="Build Status">
  <img src="assets/badges/code-quality.svg" alt="Code Quality">
  <img src="assets/badges/license.svg" alt="License">
  <img src="assets/badges/docker.svg" alt="Docker Ready">
  <img src="assets/badges/storage.svg" alt="SQLite Pure Go">
</p>

<p align="center">
  <a href="#系统架构">系统架构</a> &bull;
  <a href="#核心特性">核心特性</a> &bull;
  <a href="#快速开始">快速开始</a> &bull;
  <a href="#接入与配置流程">接入指南</a> &bull;
  <a href="docs/api.md">API 规范</a> &bull;
  <a href="AGENT.md">开发规范</a>
</p>

</div>

---

## 项目背景与解决的问题

在微信公众号（服务号/订阅号）开发与运营过程中，微信公众平台**仅允许配置一个“服务器配置 URL”**。

在实际企业级业务场景中，往往有多个独立的系统需要实时消费微信上报的事件：
- 自定义菜单系统：处理用户点击与菜单事件交互；
- 行为与合规审计系统：归档用户关注、取关、地理位置上报等全量交互日志；
- CRM / 营销系统：识别带参数关注二维码的场景值并自动分发权益；
- 客服业务系统：接收用户上行的文字、图片与语音消息。

### 传统方案对比

| 对比维度 | 传统单体代转发方案 | EventDispatcher 网关方案 |
| :--- | :--- | :--- |
| **系统耦合度** | 下游服务强依赖特定转发服务，链路冗长 | 各业务系统完全独立，按需注册订阅 |
| **单点风险** | 转发服务故障将导致全公司微信业务瘫痪 | 核心网关极简设计，内存与存储双重持久化 |
| **凭据安全** | 微信 Token/AES 密钥散落在各业务系统 | 微信通信密钥仅保存在网关，下游完全隔离 |
| **微信响应时效** | 链路多级透传，极易触发微信 5 秒超时重试 | 收到请求立即落库并回复 `success`（< 15ms） |
| **可靠性保证** | 依赖业务端自行实现重试与防重 | 网关内置全局幂等去重与指数退避重试调度 |

---

## 系统架构

```mermaid
flowchart TD
    subgraph WeChat["微信公众平台"]
        WC_Server["微信推送服务器"]
    end

    subgraph TransportLayer["传输层 (Transport / HTTP)"]
        IngestH["网关接收控制器 (IngestHandler)<br/>• 验证 URL (GET)<br/>• 接收 XML (POST)<br/>• 毫秒级回复 'success'"]
        AdminH["管理后台控制器 (AdminHandler)<br/>• 账号/订阅者管理<br/>• 审计日志/重试触发"]
    end

    subgraph ServiceLayer["领域服务层 (Application Service)"]
        IngestSvc["接入编排服务 (IngestService)<br/>• 解密与验签<br/>• 全局幂等去重<br/>• 订阅规则匹配 (Filter)"]
        DeliverySvc["投递调度服务 (DeliveryService)<br/>• 后台抢占式 Worker<br/>• HMAC-SHA256 出站签名<br/>• 阶梯指数退避重试"]
        AdminSvc["配置管理服务 (Account/Subscriber Service)"]
    end

    subgraph InfraLayer["基础设施与持久化 (Infrastructure)"]
        WeChatAdapter["微信协议适配器<br/>• SHA1 验签<br/>• AES-256-CBC 逆填充"]
        Storage[("嵌入式 SQLite (WAL 模式)<br/>• 单连接写隔离<br/>• 事件与投递状态机持久化")]
    end

    subgraph Subscribers["下游业务系统 (Webhooks)"]
        Sub1["菜单业务服务 (只收 CLICK)"]
        Sub2["用户画像/审计系统 (收全量事件)"]
        Sub3["外部 CRM / 客服系统 (收 text / scan)"]
    end

    WC_Server -->|HTTPS 报文| IngestH
    IngestH --> IngestSvc
    AdminH --> AdminSvc
    IngestSvc --> WeChatAdapter
    IngestSvc --> Storage
    AdminSvc --> Storage
    Storage --> DeliverySvc
    DeliverySvc -->|POST JSON + HMAC 签名| Sub1
    DeliverySvc -->|POST JSON + HMAC 签名| Sub2
    DeliverySvc -->|POST JSON + HMAC 签名| Sub3
```

---

## 核心特性

- **完整支持微信加解密协议**：支持明文模式、兼容模式与安全模式（AES-256-CBC），自动完成 PKCS#7 逆填充与消息解析。
- **高吞吐与极速响应**：接收到微信推送后即刻持久化并输出 `success`（典型处理耗时小于 15ms），下游投递完全异步化。
- **灵活的路由过滤机制**：各订阅方可独立配置过滤规则（`msgTypes`、`events`、`eventKeyPrefixes`），精准分发所需事件。
- **可靠的阶梯重试调度**：下游返回非 2xx 或请求超时时，自动按阶梯间隔退避重试（最多 6 次，间隔约 1s、5s、30s、2m、10m），支持手动触发重放。
- **企业级架构与安全设计**：
  - 遵循标准 Clean Architecture 分层，业务逻辑与外部框架彻底解耦；
  - 管理后台接口采用 Bearer Token 恒定时间比较（Constant Time Compare），抵御时序攻击；
  - 微信密钥与订阅 Secret 在查询接口中完全脱敏，仅返回设置状态布尔值，杜绝明文回显；
  - 下游投递采用标准的 HMAC-SHA256 签名算法；
  - 全链路 TraceID 追踪贯穿。
- **零外部中间件依赖**：基于纯 Go 驱动的 SQLite（无 CGO），单个二进制文件即可运行，极简运维与备份。

---

## 快速开始

### 方式一：源码编译与运行

#### 环境要求
- Go 1.22 或更高版本

#### 操作步骤
```bash
# 1. 克隆代码库
git clone https://github.com/your-org/EventDispatcher.git
cd EventDispatcher

# 2. 设置环境变量并启动
export ADMIN_TOKEN="your-secure-admin-token"
export ADDR=":8080"
export DB_PATH="data/dispatcher.db"

go run ./cmd/dispatcher
```

### 方式二：Docker 容器化部署

```bash
docker compose up -d
```

默认数据持久化目录映射至当前目录的 `./data`。

---

## 配置说明

系统参数通过环境变量配置，业务参数通过数据接口持久化管理：

| 环境变量 | 必填 | 默认值 | 说明 |
| :--- | :---: | :---: | :--- |
| `ADMIN_TOKEN` | 是 | - | 管理后台接口鉴权口令（Bearer Token） |
| `ADDR` | 否 | `:8080` | HTTP 服务监听地址与端口 |
| `DB_PATH` | 否 | `data/dispatcher.db` | SQLite 数据库文件存放路径 |

---

## 接入与配置流程

### 1. 登记微信服务号

```bash
curl -X POST http://127.0.0.1:8080/admin/accounts \
  -H "Authorization: Bearer your-secure-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "企业官方服务号",
    "accountKey": "main",
    "appId": "wx1234567890abcdef",
    "token": "your-wechat-token",
    "encodingAesKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
  }'
```
记录返回结果中 `data.id` 字段（如 `acc_01JBN...`）。

### 2. 在微信公众平台后台配置服务器地址

前往微信公众平台后台 -> **设置与开发** -> **基本配置** -> **服务器配置**：
- **URL**：`https://<你的域名>/w/main`
- **Token**：与登记的 `token` 一致
- **EncodingAESKey**：与登记的 `encodingAesKey` 一致
- **加解密模式**：支持明文模式、兼容模式或安全模式

提交后，EventDispatcher 会自动响应 `GET /w/main` 校验请求并回显 `echostr`。

### 3. 配置下游订阅方

#### 示例 A：菜单交互系统（仅接收菜单点击事件）
```bash
curl -X POST http://127.0.0.1:8080/admin/subscribers \
  -H "Authorization: Bearer your-secure-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "菜单业务程序",
    "accountId": "acc_你的账号ID",
    "callbackUrl": "https://service.internal/webhook/menu",
    "secret": "menu-service-secret",
    "filter": {
      "msgTypes": ["event"],
      "events": ["CLICK"],
      "eventKeyPrefixes": ["MENU_"]
    }
  }'
```

#### 示例 B：全量事件审计系统（接收全部消息与事件）
```bash
curl -X POST http://127.0.0.1:8080/admin/subscribers \
  -H "Authorization: Bearer your-secure-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "审计归档程序",
    "accountId": "acc_你的账号ID",
    "callbackUrl": "https://audit.internal/webhook/audit",
    "secret": "audit-service-secret"
  }'
```

---

## 下游系统接收规范与验签

分发中心通过 HTTP `POST` JSON 请求向订阅方投递事件。

### 请求头信息
```http
Content-Type: application/json
User-Agent: event-dispatcher/1
X-Dispatcher-Event-Id: evt_01JBN6G...
X-Dispatcher-Timestamp: 1730874600
X-Dispatcher-Signature: 8b072d9e...
```

### 请求体结构
```json
{
  "id": "evt_01JBN6G...",
  "accountKey": "main",
  "appId": "wx1234567890abcdef",
  "msgType": "event",
  "event": "CLICK",
  "eventKey": "MENU_HELP",
  "fromUser": "oXv6u0-dummy-openid",
  "toUser": "gh_123456789abc",
  "createTime": 1730874590,
  "receivedAt": "2026-11-06T06:29:50Z",
  "fields": {
    "ToUserName": "gh_123456789abc",
    "FromUserName": "oXv6u0-dummy-openid",
    "CreateTime": "1730874590",
    "MsgType": "event",
    "Event": "CLICK",
    "EventKey": "MENU_HELP"
  }
}
```

### 签名计算公式
$$\text{Signature} = \text{hex}(\text{HMAC-SHA256}(\text{secret}, \text{timestamp} + \text{"."} + \text{rawBody}))$$

> 验签时须直接使用接收到的**原始字节流（Raw Body）**，禁止反序列化后再重新序列化。完整的多语言（Go、Python、Java、Node.js、PHP）实现代码请参考 [接口文档 (docs/api.md)](docs/api.md)。

---

## 运维与故障排查

### 1. 查询事件接收记录
```bash
curl -s http://127.0.0.1:8080/admin/events \
  -H "Authorization: Bearer your-secure-admin-token"
```

### 2. 检索投递失败的任务
```bash
curl -s "http://127.0.0.1:8080/admin/deliveries?status=failed" \
  -H "Authorization: Bearer your-secure-admin-token"
```

### 3. 手动触发失败任务重试
```bash
curl -X POST http://127.0.0.1:8080/admin/deliveries/dlv_目标ID/retry \
  -H "Authorization: Bearer your-secure-admin-token"
```

### 4. 服务健康检查
```bash
curl -s http://127.0.0.1:8080/healthz
# 正常响应: {"code":0,"message":"success","data":{"status":"ok"},"traceId":"..."}
```

---

## 相关文档

- **[对外接口文档 (docs/api.md)](docs/api.md)**：包含完整的 Admin API 规范、下游 Webhook 详细定义及各语言 SDK 级验签示例。
- **[开发者与 AI 协作规范 (AGENT.md)](AGENT.md)**：包含企业级整洁架构分层规范、注释与代码风格、SQLite 数据库并发军规、Git 提交与质量保障基线。

---

## 许可证

本项目采用 [MIT 许可证](LICENSE) 开源。
