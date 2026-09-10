#!/usr/bin/env bash
# 契约: 记忆端点全形状 (doc-02 §2)。infer=false 确定性; PUT 部分更新三用例 (红线 T1);
# GET /memories 双模式分别快照 (T6); search 顶层 id 兼容 (T8)。

suite() {
  local AH_UKEY="X-API-Key: $USER_API_KEY"
  local AH_AKEY="X-API-Key: $ADMIN_API_KEY"
  local AH_JWT="Authorization: Bearer $ACCESS_TOKEN"

  # ---------- 写入 (infer=false, stub embedder 确定性) ----------
  http POST /memories '{"messages":[{"role":"user","content":"Alice likes hiking in the mountains."}],"user_id":"alice","infer":false,"metadata":{"pref":"hiking","source":"contract"}}' "$AH_UKEY"
  check_status "add alice-hiking" 200
  check_json "event=ADD" '.results[0].event' 'ADD'
  g "add-alice-hiking"
  M1=$(printf '%s' "$R_BODY" | jq -r '.results[0].id')

  # 同内容重复 → hash 去重 → NONE
  http POST /memories '{"messages":[{"role":"user","content":"Alice likes hiking in the mountains."}],"user_id":"alice","infer":false}' "$AH_UKEY"
  check_status "add 重复" 200
  check_json "重复 → 仍 ADD (infer=false 无 hash 去重)" '.results[0].event' 'ADD'
  g "add-duplicate-none"

  # 带过期日
  http POST /memories '{"messages":[{"role":"user","content":"Alice enjoys swimming in lakes."}],"user_id":"alice","infer":false,"metadata":{"pref":"swimming"},"expiration_date":"2099-01-01"}' "$AH_UKEY"
  check_status "add alice-swimming(exp)" 200
  M2=$(printf '%s' "$R_BODY" | jq -r '.results[0].id')

  # 已过期
  http POST /memories '{"messages":[{"role":"user","content":"Alice had a dog in 2015."}],"user_id":"alice","infer":false,"expiration_date":"2020-01-01"}' "$AH_UKEY"
  check_status "add alice-dog(expired)" 200
  M3=$(printf '%s' "$R_BODY" | jq -r '.results[0].id')

  # agent 记忆 + actor/role/attributed_to metadata (锁定模式A/B 形状分歧, 红线 T6)
  http POST /memories '{"messages":[{"role":"user","content":"Carol drives an old van."}],"agent_id":"agent-carol","infer":false,"metadata":{"actor_id":"act-9","role":"user","attributed_to":"a-1","custom":"c1"}}' "$AH_UKEY"
  check_status "add agent-carol" 200
  M4=$(printf '%s' "$R_BODY" | jq -r '.results[0].id')

  http POST /memories '{"messages":[{"role":"user","content":"Bob speaks French fluently."}],"user_id":"bob","infer":false,"metadata":{"lang":"fr"}}' "$AH_UKEY"
  check_status "add bob" 200
  M5=$(printf '%s' "$R_BODY" | jq -r '.results[0].id')

  # ---------- PUT 部分更新 (红线 T1: model_fields_set 语义) ----------
  # 只传 metadata → 内容不动
  http PUT "/memories/$M1" '{"metadata":{"pref":"trail-running"}}' "$AH_UKEY"
  check_status "PUT 只传 metadata" 200
  g "put-metadata-only"
  http GET "/memories/$M1" "$AH_UKEY"
  check_status "GET M1 after metadata update" 200
  check_json "内容未动" '.memory' 'Alice likes hiking in the mountains.'
  check_json "metadata 已换" '.metadata.pref' 'trail-running'

  # 显式清除 expiration_date
  http PUT "/memories/$M2" '{"expiration_date": null}' "$AH_UKEY"
  check_status "PUT expiration null 清除" 200
  http GET "/memories/$M2" "$AH_UKEY"
  check_json "expiration 已清" '.expiration_date' 'null'
  check_json "内容未动(M2)" '.memory' 'Alice enjoys swimming in lakes.'
  check_json "metadata 保留" '.metadata.pref' 'swimming'

  # 显式 text null: SDK update(data=None) 拒绝 → 400 (doc-02 §2.4 "显式置空"与实测不符, 以实测为准)
  http PUT "/memories/$M2" '{"text": null}' "$AH_UKEY"
  check_status "PUT text null → 400" 400
  g "put-text-null-400"
  http GET "/memories/$M2" "$AH_UKEY"
  g "put-text-null-result"

  # 空对象 (fields_set 空) → 400
  http PUT "/memories/$M2" '{}' "$AH_UKEY"
  check_status "PUT 空对象 → 400" 400
  check_json "空对象文案" '.detail' 'At least one of text, metadata, or expiration_date must be provided.'

  # text 更新 → 内容重写
  http PUT "/memories/$M1" '{"text":"Alice loves trail running now."}' "$AH_UKEY"
  check_status "PUT text 更新" 200
  http GET "/memories/$M1" "$AH_UKEY"
  check_json "text 已换" '.memory' 'Alice loves trail running now.'

  # ---------- GET 双模式 (T6) ----------
  http GET "/memories?user_id=alice&top_k=100" "$AH_UKEY"
  check_status "模式A: user_id=alice" 200
  check_absent "模式A: 过期记忆隐藏" '"Alice had a dog in 2015."'
  check_contains "模式A: 含未过期" 'Alice enjoys swimming in lakes.'
  g "get-mode-a-alice"

  http GET "/memories?user_id=alice&top_k=100&show_expired=true" "$AH_UKEY"
  check_status "模式A: show_expired" 200
  check_contains "模式A: show_expired 含过期" 'Alice had a dog in 2015.'

  http GET "/memories?top_k=1000" "$AH_AKEY"
  check_status "模式B: admin 全量" 200
  g "get-mode-b-all"

  # 403 分支 (role!=admin 且非 admin 类凭据) 在单 admin 拓扑不可达 → Go 单测覆盖

  # 单条 / 不存在
  http GET "/memories/$M1" "$AH_UKEY"
  check_status "GET 单条" 200
  g "get-single"
  http GET "/memories/00000000-0000-0000-0000-000000000000" "$AH_UKEY"
  check_status "GET 不存在 → null" 200
  check_json "GET 不存在 body" '.' 'null'

  # top_k 越界 422
  http GET "/memories?user_id=alice&top_k=1001" "$AH_UKEY"
  check_status "top_k=1001 → 422" 422

  # ---------- history ----------
  http GET "/memories/$M1/history" "$AH_UKEY"
  check_status "history" 200
  g "history"

  # ---------- search ----------
  http POST /search '{"query":"swimming","filters":{"user_id":"alice"},"top_k":5}' "$AH_UKEY"
  check_status "search filters" 200
  g "search-filters"

  http POST /search '{"query":"swimming","filters":{"user_id":"alice"},"top_k":5,"threshold":2.0}' "$AH_UKEY"
  check_status "search threshold=2.0 → 400 (SDK 拒绝)" 400
  g "search-threshold-400"

  http POST /search '{"query":"swimming","filters":{"user_id":"alice"},"top_k":5,"explain":true}' "$AH_UKEY"
  check_status "search explain" 200
  g "search-explain"

  # 顶层 deprecated id 仍工作 (T8)
  http POST /search '{"query":"swimming","user_id":"alice","top_k":5}' "$AH_UKEY"
  check_status "search 顶层 user_id (deprecated)" 200
  check_contains "顶层 id 有结果" 'Alice enjoys swimming in lakes.'
  g "search-toplevel-deprecated"

  # ---------- DELETE ----------
  http DELETE "/memories/$M5" "$AH_UKEY"
  check_status "DELETE 单条" 200
  check_json "删除文案" '.message' 'Memory deleted successfully'
  g "delete-single"
  http GET "/memories/$M5" "$AH_UKEY"
  check_json "删除后 GET → null" '.' 'null'

  http DELETE "/memories?user_id=bob" "$AH_AKEY"
  check_status "DELETE 按实体 (admin)" 200
  check_json "批量删除文案" '.message' 'All relevant memories deleted'
  g "delete-all-by-user"

  http DELETE "/memories" "$AH_AKEY"
  check_status "DELETE 无参 → 400" 400
  check_json "400 文案" '.detail' 'At least one identifier is required.'

  http DELETE "/memories?user_id=alice" "$AH_UKEY"
  check_status "DELETE 按实体 (user key, admin 角色) → 200" 200

  # 不存在 → 404 (ValueError "not found" 映射)
  http DELETE "/memories/11111111-2222-3333-4444-555555555555" "$AH_UKEY"
  check_status "DELETE 不存在 → 404" 404
  check_contains "404 文案含 not found" 'not found'
}
