# Python CLI 开发指南

## 环境与检查

Python 最低 3.10，运行依赖为 Typer、Rich、httpx。`uv.lock` 固定开发及 CI 依赖；所有命令在 `cli/python/` 执行。

```bash
uv sync --frozen --extra dev
uv run --frozen ruff check .
uv run --frozen ruff format --check .
uv run --frozen python scripts/check_docs.py
uv run --frozen mypy
uv run --frozen pytest tests -q
uv build
```

Ruff 使用 Google docstring 约定，检查模块、类、函数文档及格式。摘要和过程注释必须中文；复杂约束补充 `Args:`、`Returns:`、`Raises:`，简单参数依靠严格类型标注，免于重复描述。D415 不识别中文句号，由 `check_docs.py` 检查中文摘要和句末标点；D417 不要求所有简单参数都重复写说明。测试函数可不写 docstring，已有注释仍必须中文。行宽 100，双引号，目标语法 `py310`。

mypy 对全部产品源码启用 strict；外部 JSON 经过运行时校验。Typer 声明式选项保留默认值以兼容命令合同，业务函数参数显式传入。可变输入不原地改写，配置更新返回新值或保存独立副本。

## 代码职责

| 目录 | 职责 |
| --- | --- |
| `cli/program.py`、`cli/registrations/` | 装配 Typer 命令树、声明参数和稳定帮助 |
| `cli/commands/` | 命令执行、输出与用户确认 |
| `application/` | 实体范围、过期日期及初始化用例 |
| `backend/` | 抽象端口、工厂、Platform 和认证 HTTP 协议、响应校验 |
| `config/` | 类型模型、环境优先级、纯配置更新和文件保存 |
| `output/` | 品牌面板、文本和 JSON 格式 |
| `runtime/` | 单次调用状态、资源清理、终端输入和告警 |
| `integrations/` | 遥测、代理环境识别、已有插件密钥同步 |

`app.py` 保留稳定导出，`cli/run.py` 负责进程参数和最终错误退出。每次调用建立独立 ContextVar 状态，退出时关闭连接并恢复外层上下文。Backend 通过注入获取调用方身份和平台通知接收函数，不导入 Typer 或命令状态。

添加功能时，先更新需求和当日计划，再在对应命令模块实现；只有需要新增协议能力时扩展 Backend。参数和英文帮助放入注册模块，中文 docstring 说明实现；不能让 Typer 自动把中文内部文档变成帮助文案。用真实本地 HTTP 测试核对请求路径、字段、输出和失败退出码。

## 安装包验收与 CI

根目录 `.github/workflows/ci.yml` 的 Python job 覆盖 3.10–3.14：冻结安装、构建 sdist/wheel，在另一虚拟环境安装实际 wheel 后运行完整测试。3.13 另执行 Ruff、中文注释和 mypy。根目录 `make cli-py-test` 使用项目 uv 环境运行完整测试。

本地可用临时目录安装 wheel，避免被源码可编辑安装掩盖打包遗漏：

```bash
uv build
package_check_dir=$(mktemp -d)
uv venv "$package_check_dir/venv"
uv pip install --python "$package_check_dir/venv/bin/python" dist/*.whl pytest
package_tests_dir="$PWD/tests"
(cd "$package_check_dir" && "$package_check_dir/venv/bin/python" -m pytest "$package_tests_dir" -q)
```

`test_http_integration.py` 使用本地 HTTP 服务验证 CRUD、响应校验、初始化/认领、配置权限、遥测和安装入口。`test_runtime_integration.py` 验证多次调用隔离以及 POSIX 终端密钥遮罩、编辑、中断和终端恢复。测试隔离 HOME、插件路径和遥测，不访问真实平台或用户插件配置。

本地模拟协议不等同真实 SaaS、邮件送达或代理客户端验收；Windows 终端和远程 CI 需分别验证。原 `[oss]` extra 引用无法解析且未使用的 `memgoai`，已移除；当前没有 Python OSS 连接器。迁移兼容边界、验收结论与回滚范围见 [需求](../../docs/REQUIREMENTS.md#python-cli-改造) 和 [当日计划](../../docs/PLAN_md/PLAN_260911.md#python-cli-改造)。
