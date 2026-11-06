# 微信公众平台接入与配置指南

本文档介绍如何将一个或多个微信公众号（服务号/订阅号）接入到 EventDispatcher 网关。

---

## 接入前准备

在配置微信后台之前，您需要准备好：
1. 部署运行的 EventDispatcher 服务，并具备可公开访问的 HTTPS 域名（本地开发可使用 frp、cpolar 或 ngrok 等内网穿透工具）。
2. 微信公众平台管理员权限。
3. EventDispatcher 的管理员鉴权口令（`ADMIN_TOKEN`）。

---

## 第一步：在 EventDispatcher 登记服务号

可通过 Web 控制台（推荐）或 REST API 进行登记：

### 方式 A：通过 Web 控制台登记
1. 浏览器打开 `http://<your-host>:8080/ui/#/accounts`。
2. 输入管理员口令后，点击右上角 **新增服务号**。
3. 填写以下信息：
   - **服务号名称**：便于识别的中文名称（如：企业官方服务号）。
   - **路由标识 Key**：唯一的英文 URL 路径代号（如：`main`、`mall_mp`）。
   - **开发者 AppID**：微信后台开发者 ID。
   - **Token**：自行约定的验证令牌（如：`wechat_sec_token_99`）。
   - **EncodingAESKey**：43 位消息加密密钥（留空则为明文模式）。
4. 保存后，在列表中点击 **复制 URL**，获取完整的 Webhook 接入地址：
   `https://<你的域名>/w/<accountKey>`

### 方式 B：通过 REST API 登记
```bash
curl -X POST http://127.0.0.1:8080/admin/accounts \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "企业官方服务号",
    "accountKey": "main",
    "appId": "wx1234567890abcdef",
    "token": "your-wechat-token",
    "encodingAesKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
  }'
```

---

## 第二步：在微信公众平台配置服务器地址

1. 登录 [微信公众平台后台](https://mp.weixin.qq.com/)。
2. 导航至左侧菜单：**设置与开发** -> **基本配置** -> **服务器配置**。
3. 点击 **修改配置**，填写如下参数：
   - **URL**：填入第一步复制的地址，形如 `https://<你的域名>/w/main`。
   - **Token**：填写第一步中登记的 `token`。
   - **EncodingAESKey**：填写第一步中登记的 `encodingAesKey`。
   - **消息加解密方式**：支持 **明文模式**、**兼容模式** 或 **安全模式**（推荐安全模式）。
4. 点击 **提交**：
   微信服务器将向该 URL 发送 `GET` 请求进行签名合法性验证。EventDispatcher 会自动计算 SHA-1 签名，并在校验通过后毫秒级原样回显 `echostr`。
5. 验证成功后，点击 **启用** 服务器配置。

---

## 第三步：多公众号同时接入与隔离

EventDispatcher 天然支持多公众号隔离：
- 为每个公众号分配独立的 `accountKey`（如 `/w/retail`、`/w/support`）。
- 每个公众号拥有独立的 Token 和 EncodingAESKey，互不干扰。
- 下游订阅者在创建时绑定对应的 `accountId`，即可精准只消费特定公众号的事件。
