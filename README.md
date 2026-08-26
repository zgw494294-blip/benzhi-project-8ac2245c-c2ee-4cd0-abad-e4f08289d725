# cleanroom-release-go

`cleanroom-release-go` 是面向洁净室环境管理团队的偏差调查与复产放行服务。它把受控点位建档、采样方案锁定、监测结果判定、超限调查、纠正措施、验证复采、质量审核、冻结快照和放行凭据串成一条可审计状态流程。

服务只使用本地文件，不依赖外部数据库或第三方系统。写命令必须提交 `idempotencyKey`、`expectedVersion` 和 `actor`；同一幂等键用于不同请求会被拒绝，陈旧版本会返回冲突。冻结后的业务数据不可修改。

## 构建、运行与测试

```text
go build ./cmd/cleanzone
go run ./cmd/cleanzone -addr=127.0.0.1:19081 -data=./data/ledger
go test ./...
```

默认监听 `127.0.0.1:19081`，可通过 `-addr=127.0.0.1:<port>` 配置完整地址。如果没有显式传入 `-addr`，也可通过 `PORT` 提供端口号，此时固定绑定到 `127.0.0.1:<PORT>`。服务拒绝 `0.0.0.0`、`::` 和省略主机的监听地址。

`-data` 指定本地数据目录，默认为 `./data/ledger`。目录中包含带校验链的 `events.jsonl` 和可原子替换的 `projection.json`。投影缺失或损坏时会从事件账本重建；事件序号、前序摘要或校验和异常时服务拒绝启动。

运行完整 HTTP 冒烟流程并自动退出：

```text
go run ./cmd/cleanzone -selfcheck -addr=127.0.0.1:19081
```

selfcheck 会启动真实监听服务，依次模拟计划采样超限、调查草稿分次维护、批量纠正、首次验证再次超限并退回、第二次验证合格、审核退回后通过、冻结签发和公开核验；最后读取周期台账、采样矩阵、措施清单、预检与验证对比，完成后优雅关闭。

## API 概览

所有业务 API 使用 JSON，版本前缀为 `/api/v1`。

- `POST /api/v1/campaigns`：创建周期并登记受控监测点。
- `GET /api/v1/campaigns`：按 `facilityName`、`status`、`createdFrom`、`createdTo` 组合筛选，并使用 `pageSize`、`cursor` 稳定分页。
- `GET /api/v1/campaigns/statistics`：按相同条件汇总状态数量和调查、纠正、验证、审核待办。
- `POST /api/v1/campaigns/{campaignID}/plan/lock`：锁定方案及摘要。
- `POST /api/v1/campaigns/{campaignID}/observations/planned`：记录计划采样并自动判定。
- `GET /api/v1/campaigns/{campaignID}/sampling/progress`：查询轮次乘点位覆盖矩阵、完成比例和超限工作清单；可按 `round`、`area`、`metric` 筛选。
- `PATCH /api/v1/campaigns/{campaignID}/investigations/{investigationID}/draft`：部分更新影响范围、原因假设和证据引用。
- `GET /api/v1/campaigns/{campaignID}/investigations/{investigationID}/preflight`：只读检查调查闭合材料。
- `POST /api/v1/campaigns/{campaignID}/investigations/{investigationID}/conclude`：补齐证据并确认根因。
- `POST /api/v1/campaigns/{campaignID}/corrective-actions`：登记纠正措施。
- `POST /api/v1/campaigns/{campaignID}/corrective-actions/{actionID}/complete`：提交完成证据。
- `POST /api/v1/campaigns/{campaignID}/corrective-actions/batch`：原子批量登记纠正措施，单批最多 100 项。
- `POST /api/v1/campaigns/{campaignID}/corrective-actions/batch-complete`：原子批量提交措施完成证据，单批最多 100 项。
- `GET /api/v1/campaigns/{campaignID}/corrective-actions`：按 `investigationId`、`owner`、`status`/`completed` 和 `overdue` 查询措施及汇总。
- `POST /api/v1/campaigns/{campaignID}/verifications`：开始下一次验证轮次。
- `GET /api/v1/campaigns/{campaignID}/verifications/preflight`：结构化返回验证开轮阻断项。
- `GET /api/v1/campaigns/{campaignID}/verifications/comparison`：按 `fromRound`、`toRound` 查询原点位跨轮结果与轮次结论。
- `POST /api/v1/campaigns/{campaignID}/observations/verification`：记录验证结果；再次超限会自动建立新调查并回退。
- `POST /api/v1/campaigns/{campaignID}/reviews`：作出 `reject` 或 `approve` 决定；通过时冻结快照。
- `POST /api/v1/campaigns/{campaignID}/credentials`：为冻结周期签发凭据。
- `GET /api/v1/public/credentials/{credentialID}/verify`：公开核验签名与冻结摘要。
- `GET /api/v1/campaigns/{campaignID}` 与 `GET /api/v1/campaigns/{campaignID}/audit`：查询状态和审核轨迹。
- `GET /healthz`：进程健康检查。

错误响应统一包含 `error.code` 和 `error.message`。校验错误返回 `400`，资源不存在返回 `404`，版本、幂等、非法状态及冻结冲突返回 `409`。
