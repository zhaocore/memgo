# MemGo OSS HTTP API 参考

MemGo server 的自托管 REST API。Base URL:`https://memgo.wxget.com`(env `MEMGO_BASE_URL` 覆盖)。

所有端点要求请求头:`X-API-Key: <key>`(不是 `Authorization: Token`)。

## 端点总览

| 操作 | Method | URL |
|------|--------|-----|
| 写入记忆 | `POST` | `/memories` |
| 搜索记忆 | `POST` | `/search` |
| 列表记忆 | `GET` | `/memories?user_id=&agent_id=&run_id=` |
| 取单条 | `GET` | `/memories/{id}` |
| 更新 | `PUT` | `/memories/{id}` |
| 删除单条 | `DELETE` | `/memories/{id}` |
| 批量删除 | `DELETE` | `/memories?user_id=&agent_id=&run_id=` |
| 记忆历史 | `GET` | `/memories/{id}/history` |

## 记忆对象结构

检索/列表返回的记忆对象(搜索额外带 `score`):

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 唯一标识,更新/删除用 |
| `memory` | string | 记忆文本(提取后或原样存储) |
| `user_id` | string (nullable) | 人 |
| `agent_id` | string (nullable) | 项目/仓库 |
| `run_id` | string (nullable) | 会话 |
| `hash` | string | 内容 hash,去重用 |
| `expiration_date` | string (nullable) | 过期时间(ISO 8601) |
| `metadata` | object | 自定义键值,可过滤 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 最后修改时间 |
| `score` | float (仅搜索) | 混合相关性得分,0-1 |

## 实体范围

OSS 面只有三个实体维度,无 `app_id`:

| 记忆类型 | user_id | agent_id | run_id |
|----------|---------|----------|--------|
| 项目共享记忆 | — | 仓库身份 | — |
| 个人偏好 | 本机用户 | 仓库身份 | — |
| 会话记录 | 本机用户 | 仓库身份 | 会话 id |

## 写入 POST /memories

**同步**:请求完成后记忆已落库,响应直接返回结果,无需轮询事件。

```bash
curl -X POST $MEMGO_BASE_URL/memories \
  -H "X-API-Key: $MEMGO_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "我吃素,对坚果过敏"},
      {"role": "assistant", "content": "记住了。"}
    ],
    "user_id": "alice",
    "metadata": {"source": "onboarding"}
  }'
```

Body 字段:

| 字段 | 类型 | 说明 |
|------|------|------|
| `messages` | array(必填) | `[{role, content}]`;role ∈ user/assistant/system |
| `user_id` / `agent_id` / `run_id` | string | 至少给一个,否则 400 |
| `metadata` | object | 自定义键值 |
| `infer` | boolean | `false` 原样存储不做 LLM 提取(默认 true) |
| `expiration_date` | string | 过期时间 |
| `memory_type` | string | 仅 `procedural_memory` 特判 |
| `prompt` | string | 覆盖提取提示词 |

响应(写入即完成,每条含事件):

```json
{
  "results": [
    {"id": "ea925981-...", "memory": "用户吃素,对坚果过敏。", "event": "ADD"}
  ]
}
```

完整记忆对象(含 `user_id` / `metadata` / `created_at` 等)用 `POST /search` 或
`GET /memories/{id}` 读取。

## 搜索 POST /search

```bash
curl -X POST $MEMGO_BASE_URL/search \
  -H "X-API-Key: $MEMGO_API_KEY" -H "Content-Type: application/json" \
  -d '{
    "query": "饮食偏好",
    "filters": {"user_id": "alice"},
    "top_k": 10
  }'
```

Body 字段:

| 字段 | 类型 | 说明 |
|------|------|------|
| `query` | string(必填) | 自然语言查询 |
| `filters` | object | 至少含一个实体维度,否则报错 |
| `top_k` | number | 结果数(默认 20) |
| `threshold` | number | 最低得分(默认 0.1) |
| `explain` | boolean | 附带 `score_details` |
| `show_expired` | boolean | 包含过期记忆 |

顶层 `user_id` / `agent_id` / `run_id` 也接受,会被并入 filters(已弃用,建议直接放 filters)。

响应:

```json
{
  "results": [
    {
      "id": "ea925981-...",
      "memory": "用户吃素,对坚果过敏。",
      "user_id": "alice",
      "metadata": {"source": "onboarding"},
      "created_at": "2025-03-12T12:34:56.000000+00:00",
      "score": 0.89
    }
  ]
}
```

## 过滤系统

### 简单形式

```json
{"user_id": "alice"}
```

### 逻辑操作符(根级)

根级可为 `AND`、`OR`、`NOT`:

```json
{
  "AND": [
    {"user_id": "alice"},
    {"run_id": "session_1"}
  ]
}
```

### 字段操作符

| 操作符 | 说明 |
|--------|------|
| `eq` | 等于(默认) |
| `ne` | 不等于 |
| `gt`, `gte`, `lt`, `lte` | 大于 / 大于等于 / 小于 / 小于等于 |
| `in`, `nin` | 在数组内 / 不在数组内 |
| `contains`, `icontains` | 包含(区分 / 不区分大小写) |
| `*` | 通配,匹配任意非 null 值 |

### 可过滤字段

| 字段 | 说明 |
|------|------|
| `user_id` / `agent_id` / `run_id` | `eq`, `ne`, `in`, `*` |
| `metadata.<key>` | 顶层键,`eq`, `ne`, `contains` |
| 其余 payload 键 | `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`, `contains`, `icontains`, `*` |

示例:

```json
{
  "AND": [
    {"user_id": "alice"},
    {"metadata": {"contains": "high"}}
  ]
}
```

## 列表 GET /memories

```bash
# 按范围列(常规模式,必须给至少一个实体维度)
curl "$MEMGO_BASE_URL/memories?user_id=alice&agent_id=repo-x&run_id=session_1" \
  -H "X-API-Key: $MEMGO_API_KEY"

# 不带维度 = 模式 B:仅管理员可列全部
curl "$MEMGO_BASE_URL/memories?top_k=100" -H "X-API-Key: $MEMGO_API_KEY"
```

Query 参数:`user_id` / `agent_id` / `run_id`、`top_k`(默认 20,模式 B 默认 1000)、`show_expired`。

## 取单条 / 更新 / 删除

```bash
# 取单条(不存在返回 null)
curl "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY"

# 更新(至少一个字段;text/metadata/expiration_date 均可)
curl -X PUT "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"text": "更新后的记忆", "metadata": {"verified": true}}'

# 删除单条
curl -X DELETE "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY"

# 批量删除(至少一个实体维度,不可逆)
curl -X DELETE "$MEMGO_BASE_URL/memories?user_id=alice&run_id=session_1" \
  -H "X-API-Key: $MEMGO_API_KEY"

# 历史(update/delete 审计)
curl "$MEMGO_BASE_URL/memories/<id>/history" -H "X-API-Key: $MEMGO_API_KEY"
```

## 错误与语义注意

1. **filters 缺实体维度**:`search` / `get_all` 报 400:`filters must contain at least one of: user_id, agent_id, run_id`。
2. **同步写入**:无事件轮询、无 `semantic-succeeded` / `explicitly-stored` 状态机,写入即完成。
3. **PUT 语义**:仅 `metadata` 不改内容;`expiration_date: null` 清除字段;`text: null` 返回 400;空更新返回 400。
4. **无 custom_categories**:OSS 面不做按类目分类,删掉 category 过滤。
5. **鉴权头**:`X-API-Key`,不是 `Authorization: Token`。
