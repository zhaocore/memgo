# MemGo Python 到 Go 重构计划

## 当前定位

MemGo 是 `memgo` 自托管服务的单 Go module 重构。`core/` 承载无 HTTP 依赖的记忆引擎，`server/` 承载合同、鉴权与应用库编排，`cli/go/` 是第三个 CLI，`dashboard/` 是依赖 HTTP 合同的迁入 Next.js 资产。

## 历史分期快照

| 分期 | 历史记录 | 当前处理 |
| --- | --- | --- |
| P0 | Python 基线、golden 与黑盒契约套件 | 以 `tests/contract/` 为当前可执行事实，不重采 golden。 |
| P1 | Go `core/`、provider 裁剪、prompt 生成 | 保持 provider 白名单与生成文件规则。 |
| P2 | Go HTTP 层、auth、store、迁移与静态 OpenAPI | 任何合同变更必须先更新需求与契约测试。 |
| P3 | Go CLI 与 parity golden | 未迁移项持续显式列出，不能静默扩张命令面。 |
| P4 | 部署、迁移工具、CLI OSS 冒烟和基准 | 后续部署变更按当前需求重新验证。 |
| P5 | Dashboard 迁入 | Dashboard 冒烟和自动化测试仍待补齐。 |

该计划在提交 `bf93f99` 中被删除；本文件根据该提交前的计划、当前代码与 README 重建为精简版本。表内“历史记录”不是本轮重新验证的结论。

## 已确认后续项

### 产品能力

- Go CLI：`agent-rush`、`agent-mode`、`init` 邮箱验证/`--agent` bootstrap、`plugin_sync` 与 rich 色彩面板/Spinner 未迁移。
- Go core：AsyncMemory、reranker、spaCy NER/词形还原不在范围内。
- 平台 backend：真实平台 API 的 `whoami` 与 `search` 仍需可用 API key 验证。

### 验证与质量

- Dashboard：补齐可重复的全流程冒烟；首次产品改动时建立单元测试与 Playwright，并使用稳定选择器。
- HTTP：保持 `tests/contract/` 的 Python/Go 对照；只在需求确认的上游合同变更后更新 golden。
- 部署：涉及 `deploy/`、迁移或双库拓扑时，按影响范围重新运行 Go 测试、契约栈与部署验证，不能复用历史“完成”字样。

## 变更控制

- HTTP 合同、provider 集、prompt、序列化、数据库迁移、CLI 命令矩阵与部署策略的变更，必须在 `docs/REQUIREMENTS.md`、当日计划与评审说明中记录动机、兼容性影响、验证和回滚。
- `core/prompts/prompts_gen.go` 必须由 `tools/gen_prompts.py` 生成；`tests/contract/goldens/` 仅保存已审阅的合同快照。
