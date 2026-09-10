#!/usr/bin/env bash
# 契约: 实体端点 (doc-02 §6)。聚合 (type,id) 字典序 + admin 删除。

suite() {
  local AH_UKEY="X-API-Key: $USER_API_KEY"
  local AH_AKEY="X-API-Key: $ADMIN_API_KEY"

  http GET /entities "$AH_UKEY"
  check_status "entities 列表" 200
  g "entities-list"

  # 建实体数据
  http POST /memories '{"messages":[{"role":"user","content":"Entity user note one."}],"user_id":"entity-user","infer":false}' "$AH_UKEY"
  check_status "add entity-user #1" 200
  http POST /memories '{"messages":[{"role":"user","content":"Entity user note two."}],"user_id":"entity-user","infer":false}' "$AH_UKEY"
  check_status "add entity-user #2" 200
  http POST /memories '{"messages":[{"role":"user","content":"Entity agent note."}],"agent_id":"entity-agent","infer":false}' "$AH_UKEY"
  check_status "add entity-agent" 200

  http GET /entities "$AH_UKEY"
  check_status "entities after adds" 200
  check_json "user/entity-user 计数" '[.[] | select(.type=="user" and .id=="entity-user")][0].total_memories' '2'
  check_json "agent/entity-agent 存在" '[.[] | select(.type=="agent" and .id=="entity-agent")] | length' '1'
  g "entities-after-adds"

  # admin 删除实体
  http DELETE /entities/user/entity-user "$AH_AKEY"
  check_status "DELETE entity (admin)" 200
  check_json "删除文案" '.message' 'Entity deleted'
  g "entity-delete"

  http GET /entities "$AH_UKEY"
  check_json "entity-user 已消失" '[.[] | select(.id=="entity-user")] | length' '0'


  http DELETE /entities/user/nonexistent-entity "$AH_AKEY"
  check_status "DELETE 不存在实体 → 200" 200
  check_json "文案一致" '.message' 'Entity deleted'
}
