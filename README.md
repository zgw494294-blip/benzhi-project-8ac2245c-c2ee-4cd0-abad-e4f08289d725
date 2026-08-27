# subsurface-survey-gate

`subsurface-survey-gate` 是面向市政测绘团队的地下管线探测成果质量准入服务。它把探测批次、控制基准预检、管段观测及原子批量登记、确定性质量扫描、问题工作清单、整改复扫差异、人工复核、审计时间线、冻结快照和准入凭据纳入一条可追溯流程，避免不完整或相互矛盾的成果进入建档环节。

服务提供版本化 JSON HTTP API，不依赖外部系统。每个改变状态的请求都必须携带 `expectedVersion` 和 `idempotencyKey`：前者拒绝陈旧写入，后者配合请求指纹安全返回已有响应或拒绝键冲突。批次状态依次为 `draft`、`baseline_locked`、`quality_blocked`/`ready_for_review`、`under_review`、`returned`/`approved`、`frozen`、`issued`。控制点锁定后的坐标变更只能走显式 `PATCH` 接口，形成新 `version` 和带前后值的事件；冻结后禁止修改业务资料。

## 构建、运行与测试

```text
go build ./cmd/surveygate
go run ./cmd/surveygate -addr=127.0.0.1:19081 -data=./data
go test ./...
```

默认监听 `127.0.0.1:19081`。可用 `-addr=127.0.0.1:<port>` 指定回环地址；未提供 `-addr` 时，也可把 `PORT` 设置为端口号，服务会绑定 `127.0.0.1:<PORT>`。为避免意外暴露，入口拒绝非回环地址。

运行会自动结束的真实 HTTP 自检：

```text
go run ./cmd/surveygate -selfcheck -addr=127.0.0.1:19081
```

自检使用临时数据目录，启动真实回环监听，通过公开 API 完成创建批次、登记并锁定两个控制点、登记管段、扫描、提交复核、批准、冻结、签发与核验，然后优雅停机。

## 数据目录

`-data` 默认指向 `./data`。`events.jsonl` 是追加式事件账本，每条记录包含单调序号、前序摘要、当前摘要、领域事件、幂等响应和可恢复投影。`snapshots/<campaignId>.json` 使用同目录临时文件、`fsync` 与原子替换生成，并包含 `schemaVersion`、链根和状态摘要。启动时会逐行校验事件摘要链并恢复批次；链条损坏时拒绝启动。

## API 概览

- `GET /healthz`：健康检查。
- `POST /api/v1/campaigns`、`GET /api/v1/campaigns/{campaignID}`：创建和查询批次。
- `POST /api/v1/campaigns/{campaignID}/controls`：登记控制点。
- `PATCH /api/v1/campaigns/{campaignID}/controls/{controlID}`：显式修订控制点并保留前后值。
- `GET /api/v1/campaigns/{campaignID}/baseline/readiness`：只读检查草稿基准的控制点数量、核验信息、闭合风险、坐标范围、最小点间距和稳定摘要。
- `POST /api/v1/campaigns/{campaignID}/baseline/lock`：锁定至少两个控制点组成的基准。
- `POST /api/v1/campaigns/{campaignID}/observations`：登记管段观测。
- `POST /api/v1/campaigns/{campaignID}/observations/batch`：原子登记一至一百条管段观测，整批只迁移一次版本。
- `POST /api/v1/campaigns/{campaignID}/scans`：运行稳定排序的确定性规则集。
- `GET /api/v1/campaigns/{campaignID}/issues`：组合筛选、稳定分页质量问题并返回全批次汇总。
- `GET /api/v1/campaigns/{campaignID}/scans/compare?baseScanId=...&targetScanId=...`：比较两次持久化扫描快照，返回已解决、仍存在、新增规则结果及整改证据。
- `POST /api/v1/campaigns/{campaignID}/rectifications`：提交整改说明及可选观测修订；问题只能由复扫关闭。
- `GET /api/v1/campaigns/{campaignID}/audit-events`：按账本顺序查询状态变更依据、操作者、摘要链和分页游标。
- `POST /api/v1/campaigns/{campaignID}/review/submit`、`POST /api/v1/campaigns/{campaignID}/review/decision`：提交复核、退回或批准。
- `POST /api/v1/campaigns/{campaignID}/freeze`：冻结已批准且无未解决阻断问题的快照。
- `POST /api/v1/campaigns/{campaignID}/credential`、`POST /api/v1/credentials/verify`：签发和核验准入凭据。

请求解码拒绝未知字段并限制为 1 MiB；响应统一使用 JSON。错误信封包含 `errorCode`、`message`、`requestId` 和可选 `field`。
