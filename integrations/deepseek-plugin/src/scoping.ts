/**
 * 单次调用的记忆范围。
 *
 * 插件挂载时只有一个默认 userId, 但一次 harness 安装可为多个实体服务, 所以
 * 两个工具都接受可选的 userId / agentId / runId 参数, 在单次调用中覆盖挂载
 * 默认值; 缺失或空白的参数回退到配置的用户。
 *
 * 原 SDK 版因 camel→snake 转换器, search(filters 内)与 add(顶层参数)
 * 采用不同大小写。直连 OSS 的内置客户端统一收 snake_case, 两者现在同形:
 * search 把范围展开进 `filters`, add 把字段放在顶层, 均原样传给 server。
 */
import type { Scope } from "./client.ts";

export interface EntityParams {
  userId?: string;
  agentId?: string;
  runId?: string;
}

const clean = (v: string | undefined) => v?.trim() || undefined;

/** Search: snake_case, 展开进 `filters` 原样传给 server。 */
export function resolveSearchFilters(
  params: EntityParams,
  defaultUserId: string,
): Scope {
  const scope: Scope = { user_id: clean(params.userId) ?? defaultUserId };
  const agentId = clean(params.agentId);
  if (agentId) scope.agent_id = agentId;
  const runId = clean(params.runId);
  if (runId) scope.run_id = runId;
  return scope;
}

/** Add: 与 search 同形, 仅调用侧放置位置不同(client 放在 body 顶层)。 */
export function resolveAddParams(
  params: EntityParams,
  defaultUserId: string,
): Scope {
  return resolveSearchFilters(params, defaultUserId);
}
