# 舞台吊挂安全启用工作台

这是供剧场吊挂技术负责人、技术联排监督员和独立安全评审员使用的浏览器工作台。系统将吊点载荷、换景动作、自动校核、技术联排、整改修订和最终安全评审收敛到一条可审计状态流程；评审通过后冻结当前修订摘要并生成可验证的演出启用单。

## 主要流程

1. 建立吊挂方案，录入演出场地、日期、负责人、吊点能力与换景动作。
2. 提交不可变修订并运行确定性校核，检查超载、安全系数、动作冲突和最小净空。
3. 校核通过后记录技术联排；通过结论必须附证据引用，阻断结论必须填写观察记录并提交替代修订重新校核、复验。
4. 由不同于方案提交者的独立评审员作出决定。
5. 评审通过后冻结 SHA-256 摘要、生成授权码，并可在工作台中重新验证。

## 构建与测试

项目使用 Go 和 SQLite，不需要 Node 构建链。

```bash
go build ./cmd/server
go test ./...
```

## 运行

默认仅监听高位回环地址 `127.0.0.1:19081`：

```bash
go run ./cmd/server
```

浏览器访问 `http://127.0.0.1:19081/workbench`。数据库默认写入 `rigging-workbench.db`，也可通过 `-db` 指定路径。

可显式指定监听地址：

```bash
go run ./cmd/server -addr=127.0.0.1:19181 -db=./data/rigging.db
```

未传入 `-addr` 时，合法的 `PORT` 环境变量会绑定为 `127.0.0.1:<PORT>`。显式 `-addr` 的优先级更高。

## 完整自检

selfcheck 会创建临时 SQLite 数据库，在指定地址启动真实 HTTP 服务，并依次执行建档、校核、通过联排、独立评审和授权码验证；完成后自动关闭并删除临时数据库。

```bash
go run ./cmd/server -selfcheck -selfcheck-timeout=20s -addr=127.0.0.1:19081
```

## HTTP 入口

- `GET /workbench`：原生响应式单页工作台。
- `GET|POST /api/v1/rigging-plans`：查询或建立方案。查询支持组合参数 `state`、`venue`、`performanceDateFrom`、`performanceDateTo`，并返回各状态数量统计。
- `GET /api/v1/rigging-plans/{planID}`：读取完整方案、修订差异、整改闭环和审计时间线。
- `POST /api/v1/rigging-plans/{planID}/revisions`：提交替代修订。
- `POST /api/v1/rigging-plans/{planID}/checks`：执行确定性安全校核。
- `POST /api/v1/rigging-plans/{planID}/rehearsals`：记录技术联排。
- `POST /api/v1/rigging-plans/{planID}/reviews`：提交独立评审决定。
- `GET /api/v1/authorizations/{code}`：只读复核授权状态、授权码派生关系和持久化冻结修订的 SHA-256 摘要，并返回稳定 `reason`。

写入请求可在 JSON 中提供 `requestKey`，或使用 `Idempotency-Key` 请求头。每次状态更新必须携带当前 `version`；版本过期时 API 返回 `409 VERSION_CONFLICT`，客户端应刷新后重试。
