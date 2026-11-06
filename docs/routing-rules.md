# 下游订阅与规则路由引擎

EventDispatcher 引入了声明式的 JSON 规则匹配引擎，各业务系统无需处理无关事件，仅接收符合过滤条件的事件流。

---

## 路由匹配原理

微信上报事件后，网关会提取标准化元数据：
- `msgType`：消息大类，如 `event`（事件推送）、`text`（文本消息）、`image`（图片消息）等。
- `event`：事件子类型（当 `msgType == "event"` 时生效），如 `subscribe`（关注）、`unsubscribe`（取关）、`CLICK`（菜单点击）、`SCAN`（扫码）等。
- `eventKey`：事件附带的业务标识键，如自定义菜单的 `KEY`、带参二维码场景值 `qrscene_123` 等。

每个订阅者关联一个独立的 `Filter` 规则对象。只有当前事件完全满足 Filter 中声明的非空条件时，才会生成投递任务。

---

## 过滤规则配置语法

Filter 为标准 JSON 对象，支持以下匹配字段：

| 字段 | 类型 | 说明 | 匹配逻辑 |
| :--- | :--- | :--- | :--- |
| `msgTypes` | `string[]` | 允许的消息大类列表 | 集合包含（OR）：只要命中其一即可 |
| `events` | `string[]` | 允许的事件类型列表 | 集合包含（OR）：只要命中其一即可 |
| `eventKeyPrefixes` | `string[]` | 允许的 EventKey 前缀列表 | 前缀匹配（OR）：以其中任一前缀开头即可 |

> **提示**：若某个字段留空或未配置，则表示对该维度不设限制。如果 Filter 为空对象 `{}`，则接收该服务号下的**全量事件与消息**。

---

## 常见业务场景配置示例

### 1. 自定义菜单交互系统
仅接收用户点击菜单的事件，且仅关注特定业务前缀：
```json
{
  "msgTypes": ["event"],
  "events": ["CLICK"],
  "eventKeyPrefixes": ["MENU_MALL_", "MENU_ORDER_"]
}
```

### 2. 用户增长与渠道扫码归因系统
仅接收新用户关注以及已关注用户的扫码事件：
```json
{
  "msgTypes": ["event"],
  "events": ["subscribe", "SCAN"]
}
```

### 3. 客服派单与智能问答系统
仅接收用户上行的普通对话内容（文字、图片、语音）：
```json
{
  "msgTypes": ["text", "image", "voice"]
}
```

### 4. 全量行为合规与数据仓库归档系统
接收所有消息与事件，不漏掉任何一条：
```json
{}
```
留空或配置为空对象 `{}` 即可。

---

## 出站签名与下游验签

网关在向订阅方 Webhook 发起 POST 请求时，会在请求头中附带 HMAC-SHA256 签名，防止伪造与篡改：
- `X-Dispatcher-Event-Id`：唯一事件 ID
- `X-Dispatcher-Timestamp`：当前 Unix 时间戳（秒）
- `X-Dispatcher-Signature`：`hex(HMAC-SHA256(secret, timestamp + "." + rawBody))`

详细的多语言验签实现参见 [API 接口规范文档](api.md#出站消息签名验签规范)。
