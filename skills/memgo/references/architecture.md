# MemGo 架构

MemGo 是自托管的记忆服务,单 Go module,模块路径
`github.com/zhao-core/memgo`。核心 loop:

```
用户输入 → 检索相关记忆 → 丰富 LLM prompt → 生成响应 → 存储新记忆
```

应用只调 `Search()` / `Add()`,提取、去重、冲突处理、语义检索的复杂度都在引擎内。

## 目录分层

| 路径 | 职责 |
|------|------|
| `cmd/server/` | `memgo-server` 入口:启动校验、DEFAULT_CONFIG、迁移、HTTP 装配 |
| `cli/go/` | Go CLI(cobra):命令面、Backend 接口(OSS)、config、output、telemetry |
| `core/config/` | `MemoryConfig` 解析、深合并、敏感配置脱敏 |
| `core/llm/`、`core/embedder/` | LLM/embedder 端口与 openai、anthropic、gemini 客户端 |
| `core/prompts/` | 上游 prompt 常量及消息拼装 |
| `core/memory/`、`core/entity/` | 记忆流水线(add/search/get/update/delete/history/reset)与实体抽取/加成 |
| `core/graph/` | 内存图索引(多跳检索的 entity 关系) |
| `core/vectorstore/`、`core/history/` | pgvector 实现(含过滤翻译)与 SQLite 历史库 |
| `server/` | HTTP 层:路由、鉴权、校验、middleware、错误映射、存储编排 |
| `dashboard/` | Next.js Dashboard,通过 HTTP 合同调 server |

架构规则:`core/` 是无 HTTP 依赖的记忆引擎;`server/` 只承载路由、鉴权、校验与编排,
不持有领域规则;外部系统(LLM、embedder、向量库、历史库、Postgres、遥测)都隔在端口之后。

## 双库拓扑(红线)

| 库 | 内容 |
|----|------|
| pgvector(`postgres`) | **记忆向量** + payload(JSONB) |
| `memgo_app` | users / api_keys / request_logs / jtis / settings |
| SQLite(`history.db`) | 记忆 ADD/UPDATE/DELETE 历史审计 |

记忆向量只进 pgvector;server 表只进 `memgo_app`;互不混入。

## 写入流水线(同步)

`Add()` 与上游 `Memory.add` 对齐。`infer=true` 走阶段化批量流水线:

```
消息入 → 已有记忆检索 → 单次 LLM 提取(去重+冲突解决)
       → 批量嵌入 → hash 去重(批内+既有) → 插入向量库
       → ADD 历史 → 实体联动(upsert) → 存历史消息
```

- **infer=false**:原样存储,不做 LLM 处理,跳过冲突解决。
- **同步**:整个流水线在请求内完成,响应直接返回 `results`(含 `event: "ADD"`),
  无平台的事件轮询与 `semantic-succeeded` 状态机。
- **去重**:按内容 hash(MD5 类)去重,重复事实不重复入库。

## 检索流水线

`Search()` 对齐上游 `_search_vector_store`,多信号混合评分:

```
查询入 → 预处理 + 嵌入
       → 语义检索(向量相似度, top_k*4 候选)
       → BM25 关键词检索 + sigmoid 归一化
       → 实体抽取 + 加成(命中实体记忆加权)
       → (可选)多跳图遍历加成
       → score_and_rank(合并信号 → 阈值过滤 → top_k)
```

`score` 是语义 + BM25 + 实体加成的合并值,0-1。内置集无 spaCy(lemmatize 恒等),
实体抽取依赖 `core/entity`;`graph` 提供多跳索引(`EnableMultiHop`,可关)。

## 范围与多租户

三个实体维度,无 `app_id`:

| 维度 | 字段 | 用途 |
|------|------|------|
| 人 | `user_id` | 个人偏好、长期事实 |
| 项目/仓库 | `agent_id` | 仓库级共享记忆 |
| 会话 | `run_id` | 短生命周期的会话上下文 |

仓库范围用 `agent_id` 表达(替代上游平台 `app_id`)。组合记录物理分离:同一记忆的
`user_id`+`agent_id` 组合与仅 `user_id` 的记录是不同条。

## 记忆生命周期

| 操作 | 说明 |
|------|------|
| 创建 | `Add(messages, ...)` → 单遍提取 → 去重 → 存储,同步返回 |
| 读取 | `Search` / `Get` / `GetAll` |
| 更新 | `Update(id, {text, metadata, expiration_date})`,写 UPDATE 历史 |
| 删除 | `Delete(id)` 或 `DeleteAll({user_id, agent_id, run_id})`,写 DELETE 历史 |
| 重置 | `Reset()` 清空历史库与向量库 |

`expiration_date` 设后,默认检索/列表会过滤过期记忆(`show_expired=true` 可见)。

## 性能特征

| 操作 | 说明 |
|------|------|
| add / search(stub 路径) | Go server P50 ~73ms / ~72ms(压测基线 N=50) |
| 真实路径 | 延迟由 LLM / embedder 主导,不纳入对比 |
| 批量 | add 内批量嵌入 + 批量插入 |
| 过滤 | `filters` 下推给 pgvector 的 JSONB 过滤,再用 `top_k` 限制结果 |

## 与上游托管平台的差异(迁移时注意)

1. **同步写入** —— 无事件轮询、无 `semantic-succeeded`/`explicitly-stored`。
2. **无 custom_categories** —— 不做按类目分类,删掉 category 过滤。
3. **无 app_id** —— 仓库范围用 `agent_id`。
4. **鉴权头** —— `X-API-Key`,不是 `Authorization: Token`。
5. **遥测** —— server 只落本地状态文件,不对外发送;`MEMGO_TELEMETRY=false` 关闭。
