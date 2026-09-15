# MemGo 特性

核心 CRUD 之外的记忆层能力。与上游托管平台对齐的部分保留,平台专有能力如实裁剪。

## 目录

- [同步提取](#同步提取)
- [混合评分](#混合评分)
- [实体与多跳](#实体与多跳)
- [过滤系统](#过滤系统)
- [过期日期](#过期日期)
- [历史审计](#历史审计)
- [MCP 集成](#mcp-集成)
- [已裁剪的平台特性](#已裁剪的平台特性)

## 同步提取

`add` 是全同步的:请求内完成 LLM 提取 → 去重 → 落库,响应直接返回
`{"results":[{"id","memory","event":"ADD"}]}`。写入即完成,无需轮询事件。

`infer=true`(默认)用 LLM 从对话中提取结构化事实,去重并解决冲突;
`infer=false` 原样存储,跳过 LLM,用于批量导入 / 预结构化数据。

## 混合评分

`search` 用多信号混合评分,自动开箱:

- **语义检索** —— 向量相似度(余弦,`score = max(0, 1 - distance)`)
- **BM25 关键词** —— 词项匹配,sigmoid 归一化(midpoint/steepness 随查询长度自适应)
- **实体加成** —— 命中查询实体的记忆加权

合并进每条结果的 `score`(0-1)。`threshold`(默认 0.1)过滤低分,`top_k`(默认 20)截断。
无 rerank —— reranker 未实现,`rerank` 恒 false,合同也未暴露。

```bash
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"饮食偏好","filters":{"user_id":"alice"},"top_k":10,"threshold":0.3,"explain":true}'
```

`explain=true` 时每条结果带 `score_details`。

## 实体与多跳

- **实体抽取**:`core/entity` 从记忆文本抽取实体(命名实体、引号短语等),`add` 时
  upsert 到实体 collection,`search` 时用查询实体加成相关记忆。
- **多跳图遍历**:`core/graph` 提供内存图索引,`EnableMultiHop` 开启时,`search` 沿
  entity 关系做多跳遍历并加权(`MaxHops` / `DecayFactor` 可配)。该能力是上游 v2
  graph memory 的 Go 等价,通过检索排名的加成消费,不暴露独立 `relations` 数组。
- **基线注意**:内置集无 spaCy 词法分析,抽取依赖 `core/entity`;未注入实体 collection
  时实体联动为空操作(逻辑保留以对齐行为面)。

## 过滤系统

`filters` 支持逻辑操作符与字段操作符(HTTP 与 Go `SearchParams.Filters` 同语义):

```json
{
  "AND": [
    {"user_id": "alice"},
    {"run_id": "session_1"},
    {"metadata": {"contains": "high"}}
  ]
}
```

- 根级 `AND` / `OR` / `NOT`。
- 操作符:`eq` / `ne` / `gt` / `gte` / `lt` / `lte` / `in` / `nin` / `contains` /
  `icontains` / `*`(通配,不含 null)。
- **至少一个实体维度**(`user_id` / `agent_id` / `run_id`),否则报错。

## 过期日期

`add` 或 `update` 传 `expiration_date`(ISO 8601)后,默认检索/列表过滤过期记忆;
`show_expired=true` 可见。`update` 传 `expiration_date: null` 清除该字段。

## 历史审计

每条记忆的 ADD/UPDATE/DELETE 都写进 SQLite 历史库,`GET /memories/{id}/history`(Go:
`m.History(id)`)返回 `old_memory` / `new_memory` / `event` / 时间戳 / actor。

## MCP 集成

共享 stdio MCP server(`integrations/mcp/memgo_mcp.py`)直连 OSS API,AI 客户端
(Claude Code、Codex、Cursor、Windsurf 等)可自主管理记忆:

| 工具 | 说明 |
|------|------|
| `add_memory` | 同步写入消息列表 |
| `search_memories` | 语义搜索(filters 至少一个实体维度) |
| `get_memories` | 按实体范围列记忆 |
| `get_memory` | 按 ID 取单条 |
| `update_memory` | 覆盖文本 / 元数据 |
| `delete_memory` | 删除单条 |
| `delete_all_memories` | 批量删除范围记忆 |

## 已裁剪的平台特性

以下为上游托管平台专有,OSS 面**不支持**(如实裁剪,勿在 MemGo 上使用):

| 平台特性 | 状态 | 说明 |
|----------|------|------|
| custom_categories | 裁剪 | 无按类目自动分类,无 category 过滤 |
| rerank | 裁剪 | reranker 未实现,恒 false |
| webhooks | 裁剪 | 同步写入,无异步完成通知 |
| group chat 说话人归因 | 裁剪 | 不做多参与者的实体归因 |
| multimodal(图片/PDF) | 裁剪 | 文本记忆引擎;content 数组仅拼接 text 分段 |
| memory export / 自定义导出 schema | 裁剪 | 无 `create_memory_export` |
| 平台 feedback 机制 | 裁剪 | 无 `feedback()` |
| 异步事件 / 事件轮询 | 裁剪 | 同步语义,无 event_id |

对应 SDK 参数(`custom_categories`、`rerank`、`webhook`、`feedback` 等)在接入时删掉即可。
