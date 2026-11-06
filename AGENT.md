# EventDispatcher 企业级开发治理与协作规范 (AGENT.md)

本规范是 **EventDispatcher (微信公众号事件分发中心)** 的核心工程准则与架构设计标准。无论是开发团队成员还是 AI 编程助手（如 Antigravity / Coding Assistant），在进行功能扩展、架构迭代、代码审查或重构时，均须无条件遵守本规范。

---

## 1. 架构设计与工程分层

项目采用整洁架构（Clean Architecture）与领域驱动设计（DDD）分层思想，遵循**依赖倒置原则（Dependency Inversion Principle）**：高层业务逻辑绝不直接依赖底层具体实现，所有外设（数据库、HTTP 路由、微信协议、第三方 API）均作为可替换的适配器。

```
EventDispatcher/
├── cmd/
│   └── dispatcher/               # 服务主入口：加载配置、依赖注入（DI）、启动调度与优雅停机
├── internal/
│   ├── config/                   # 系统级配置加载与校验（严格区分系统配置与业务数据）
│   ├── domain/                   # 核心领域层：纯业务实体、状态枚举、值对象及仓储契约（无外部框架依赖）
│   │   ├── account.go            # 微信公众号实体与 AccountRepository 接口契约
│   │   ├── subscriber.go         # 下游订阅方实体与 SubscriberRepository 接口契约
│   │   ├── event.go              # 标准化事件实体与 EventRepository 接口契约
│   │   ├── delivery.go           # 投递任务状态机与 DeliveryRepository 接口契约
│   │   └── filter.go             # 路由匹配领域对象与前缀过滤规则
│   ├── service/                  # 应用业务逻辑层（Use Cases）：业务编排、去重编排、退避算法与调度器
│   │   ├── account_service.go    # 账号增删查改与凭证脱敏业务
│   │   ├── subscriber_service.go # 订阅者规则校验与管理业务
│   │   ├── ingest_service.go     # 微信验签、解密、事件去重与多路广播编排
│   │   ├── delivery_service.go   # 后台抢占式 Worker、HMAC-SHA256 计算、退避重试调度
│   │   └── event_service.go      # 事件查询业务
│   ├── infrastructure/           # 基础设施与外部适配层 (Adapters)
│   │   ├── persistence/          # 存储持久化实现 (SQLite 驱动、连接池隔离、事务管理与仓储实现)
│   │   │   ├── schema.go         # 数据库 DDL 迁移定义
│   │   │   ├── sqlite.go         # 数据库连接池初始配置（WAL 模式与单连接控制）
│   │   │   ├── account_repo.go   # AccountRepository 实现
│   │   │   ├── subscriber_repo.go# SubscriberRepository 实现
│   │   │   ├── event_repo.go     # EventRepository 实现
│   │   │   └── delivery_repo.go  # DeliveryRepository 实现
│   │   └── wechat/               # 微信协议适配器：SHA1 验签、AES-256-CBC 逆填充加解密、XML 流式解析
│   └── transport/                # 传输层 (Transport / Presentation Layer)
│       └── http/                 # HTTP 协议适配
│           ├── handler/          # 控制器：参数校验、调用业务 Service、调用统一响应包装
│           │   ├── account_handler.go
│           │   ├── subscriber_handler.go
│           │   ├── event_handler.go
│           │   ├── delivery_handler.go
│           │   ├── ingest_handler.go
│           │   └── health_handler.go
│           ├── middleware/       # 传输层统一中间件
│           │   ├── trace.go      # 链路追踪（X-Trace-Id 自动生成与注入）
│           │   ├── auth.go       # 恒定时间比对 Admin Bearer Token
│           │   └── recovery.go   # Panic 捕获与降级响应
│           ├── response/         # 统一 RESTful JSON 响应体系
│           └── router.go         # 路由组装与中间件装配
├── pkg/                          # 跨项目通用的底层工具与基础能力包（绝对独立，无业务耦合）
│   ├── errcode/                  # 企业级业务错误码与错误映射规范
│   ├── id/                       # 具有业务语义前缀的防碰撞随机 ID 生成器 (acc_*, sub_*, evt_*, dlv_*)
│   └── logger/                   # 基于 log/slog 的企业级结构化日志，支持自动提取链路 TraceID
├── docs/                         # 规范文档（API 文档、架构说明）
├── Makefile                      # 常用构建与自动化工具指令
├── Dockerfile                    # 生产环境精简多阶段镜像构建
└── docker-compose.yml            # 容器化部署配置
```

### 1.1 核心分层约束
- `domain` 必须保持纯净：严禁引入 `net/http`、`database/sql` 或外部数据库驱动。
- 依赖单向流动：`transport -> service -> domain`，`infrastructure -> domain`。严禁高层直接依赖 `infrastructure` 的具体结构体，依赖注入时必须使用 `domain` 定义的接口。
- `handler` 层不写业务逻辑：Handler 只负责读取请求、基本参数反序列化、调用 Service，并使用 `response.Success` / `response.Error` 格式化输出。

---

## 2. 代码风格与编写基准

### 2.1 Go 语言规范
- 严格遵循 Go 官方 Effective Go 与 Go Code Review Comments。
- **并发与上下文**：所有 I/O、数据库操作和 HTTP 请求，必须将 `context.Context` 作为第一参数向下传递，严禁忽略上下文超时或取消信号。
- **资源清理**：对于 `http.Response.Body`、`sql.Rows`、`io.Closer`，在获取成功后必须紧随 `defer res.Close()`。
- **专有名词大小写**：结构体字段与变量中的缩写专有名词必须全大写或全小写（统一使用 `AppID`、`AccountID`、`URL`、`JSON`、`XML`、`HTTP`、`AES`，严禁出现 `AppId`、`Json`）。

### 2.2 命名规范
- **包名**：全部小写，单个单词，禁止下划线与驼峰。
- **文件命名**：全部小写下划线分割（如 `account_repo.go`、`ingest_service.go`）。
- **实体 ID 前缀约定**：所有业务主键统一由 `pkg/id` 生成，必须携带业务类型前缀：
  - `acc_`：微信账号 (Account)
  - `sub_`：订阅方 (Subscriber)
  - `evt_`：接收到的微信事件 (Event)
  - `dlv_`：出站投递任务 (Delivery)

### 2.3 注释规范 (GoDoc)
- **所有导出的包、类型、函数、常量必须具备中文 GoDoc 注释**。
- 注释必须以被注释的对象名称开头，语句简明扼要，说明其职责与边界。
  ```go
  // IngestService 处理微信网关事件接收、解密验签与广播派发。
  type IngestService struct { ... }

  // VerifyURL 处理微信首次接入验证 (GET /w/{accountKey})。
  func (s *IngestService) VerifyURL(ctx context.Context, ...) (string, error) { ... }
  ```
- **核心算法必须注明逻辑原理**：对微信 PKCS#7 逆填充、HMAC 签名哈希计算、指数退避策略等，必须保留详尽注释。

### 2.4 文档风格严正要求
- **严禁使用任何 Emoji 图标**：无论是代码注释、提交信息（Git Commit）、API 文档还是 Markdown 文档中，一律不得包含任何 Emoji 表情符号，保持严肃、专业的工业级工程标准。

---

## 3. 错误处理与响应规范

### 3.1 企业级错误码规范 (`pkg/errcode`)
项目摒弃无结构的裸字符串报错，统一采用带有 HTTP 状态码与业务错误码的 `errcode.Error` 体系：

| 错误码范围 | 类别 | 说明 | 示例 |
| :--- | :--- | :--- | :--- |
| `0` | 成功 | 操作正常完成 | `Success` |
| `40001 ~ 40099` | 参数与载荷错误 | 客户端请求数据格式或语义错误 | `ErrInvalidParam` |
| `40101 ~ 40199` | 认证授权错误 | Token 缺失、错误或失效 | `ErrUnauthorized` |
| `40401 ~ 40499` | 资源未找到 | 指定 ID 或 Key 的记录不存在 | `ErrNotFound` |
| `40901 ~ 40999` | 冲突与重名 | 唯一索引冲突或重复操作 | `ErrConflict` |
| `42001 ~ 42099` | 微信交互错误 | 微信签名错误、密文解密失败等 | `ErrWeChatBadSignature` |
| `50001 ~ 50099` | 系统内部故障 | 数据库异常、内部未捕获错误 | `ErrInternalServer` |

### 3.2 统一响应结构 (`internal/transport/http/response`)
除微信端点按官方要求返回特定文本（`success` 或 `echostr`）外，所有管理端 API 统一采用标准化 JSON 结构：
```json
{
  "code": 0,
  "message": "success",
  "data": { ... },
  "traceId": "01afc82b1d..."
}
```
- **链路追踪贯穿**：每一个请求通过 `middleware.Trace` 自动提取或生成 `X-Trace-Id`，注入上下文并在响应头与响应体中回显。
- **错误脱敏**：系统内部异常不直接暴露底层数据库报错或堆栈信息给调用方，通过日志记录全量上下文，对外仅返回脱敏后的企业标准错误响应。

---

## 4. 结构化日志规范 (`pkg/logger`)

- 统一采用 Go 标准库 `log/slog`，杜绝使用老旧的 `log.Println`。
- 必须传递上下文：优先调用 `logger.InfoContext(ctx, ...)`、`logger.ErrorContext(ctx, ...)`，以便日志 Handler 自动将 `traceId` 提取为结构化字段。
- 日志字段必须保持 KV 规范命名（使用 `slog.String`、`slog.Int` 等强类型函数）：
  ```go
  s.logger.InfoContext(ctx, "ingest event processed",
      slog.String("accountKey", acc.AccountKey),
      slog.String("msgType", msg.MsgType),
      slog.String("event", msg.Event),
      slog.Bool("duplicate", !created),
  )
  ```

---

## 5. 存储持久化与并发基线 (`internal/infrastructure/persistence`)

### 5.1 SQLite 使用军规
- **纯 Go 驱动**：使用 `modernc.org/sqlite`，无任何 CGO 依赖。
- **单连接串行化写入（核心）**：
  ```go
  rawDB.SetMaxOpenConns(1)
  rawDB.SetMaxIdleConns(1)
  rawDB.SetConnMaxLifetime(0)
  ```
  > SQLite 在多连接并发写场景下极易发生 `database is locked` 异常。**必须严格配置单连接**，配合 WAL（Write-Ahead Logging）模式，既能发挥单进程并发读的高性能，又能保证事务写严格串行安全。
- **连接字符串强制参数**：
  `?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)`

### 5.2 业务幂等与去重
- 微信消息去重策略：
  - 含有 `MsgId`：以 `<AppID>\x1f<MsgId>` 作为 `dedupe_key`；
  - 无 `MsgId`（系统事件）：以 `<AppID>\x1f<FromUserName>\x1f<CreateTime>\x1f<MsgType>\x1f<Event>\x1f<EventKey>` 作为 `dedupe_key`。
- `events` 表中设置 `dedupe_key UNIQUE` 约束，入库时采用 `ON CONFLICT(dedupe_key) DO NOTHING`。若已存在则直接返回成功但不重复生成出站投递任务。

---

## 6. 安全基线

1. **凭证脱敏原则**：
   - 微信 `Token`、`EncodingAESKey` 以及订阅方的 `Secret` 属于高敏凭据。
   - 所有查询接口（列表或详情）**严禁回显明文密钥**，只返回布尔状态字段（`tokenSet`、`encodingAesKeySet`、`secretSet`）。
   - 更新接口遵循“按需覆盖”原则：若请求体中未传递新密钥，则自动保留原凭据，防止更新时误清空。
2. **时序攻击防御**：
   - 对 Admin Bearer Token 验证、微信 SHA1 签名比对，必须使用 `crypto/subtle.ConstantTimeCompare`，杜绝利用对比耗时差异进行侧信道攻击。
3. **下游 Webhook 验签防篡改**：
   - 投递下游使用 `hex(HMAC-SHA256(secret, timestamp + "." + rawBody))`。
   - 下游系统必须直接消费原始网络字节流进行验签，严禁反序列化后再序列化。

---

## 7. 测试与工程质量基线

- **自动化测试规范**：
  - 领域规则（如 `filter_test.go`）必须具备 100% 分支覆盖率的表驱动测试；
  - 投递调度与退避策略（如 `delivery_service_test.go`）必须覆盖各轮次间隔与签名算法；
  - 关键链路必须包含端到端集成测试（`integration_test.go`），在真实 SQLite 内存/临时库中验证账号创建、微信验签、事件接收与 Webhook 广播全流程。
- **测试通过要求**：任何提交前，必须运行以下命令并确保 100% 成功通过：
  ```bash
  go test -v ./...
  ```

---

## 8. Git 提交规范 (Conventional Commits)

提交说明必须遵循 Conventional Commits 格式，严禁出现任何 Emoji 图标：
`<type>(<scope>): <subject>`

- **type**：
  - `feat`: 新增特性或接口
  - `fix`: 修复缺陷或逻辑异常
  - `refactor`: 重构（不改变现有行为的代码结构调整）
  - `perf`: 性能优化
  - `test`: 增加或修正测试用例
  - `docs`: 文档变更
  - `chore`: 构建配置、依赖版本更新等
- **scope** 范例：`account`, `subscriber`, `ingest`, `delivery`, `persistence`, `http`
- **示例**：
  - `feat(ingest): 支持安全模式下微信消息自动解密`
  - `fix(delivery): 修正最大重试次数达到后状态未流转为 failed 的问题`
  - `refactor(transport): 将控制器拆分为独立业务 Handler`

### 8.1 提交与推送权限准则 (核心协作铁律)
- **严禁自主提交**：AI 助手在完成任何代码、配置或文档修改后，**绝对严禁**擅自执行 `git commit` 或 `git push`；
- **用户指令驱动**：所有变更必须停留在本地工作区，向用户清晰汇报变更内容与测试结果，**只有在用户明确下达提交指令后，方可执行 git 提交与远程推送**。
