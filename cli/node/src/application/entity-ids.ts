import type { MemGoConfig } from '../config/types.js';
/**
 * 优先使用显式实体 ID；未指定任何 ID 时读取配置默认值。
 * @param config - 当前配置。
 * @param opts - 本次操作选项。
 * @param opts.userId - 用户 ID。
 * @param opts.agentId - 代理 ID。
 * @param opts.appId - 应用 ID。
 * @param opts.runId - 会话 ID。
 * @returns 本次调用使用的实体 ID。
 */
export function resolveIds(
  config: MemGoConfig,
  opts: {
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
  }
): { userId?: string; agentId?: string; appId?: string; runId?: string } {
  const hasExplicit = !!(opts.userId || opts.agentId || opts.appId || opts.runId);
  if (hasExplicit) {
    return {
      userId: opts.userId || undefined,
      agentId: opts.agentId || undefined,
      appId: opts.appId || undefined,
      runId: opts.runId || undefined,
    };
  }
  return {
    userId: config.defaults.userId || undefined,
    agentId: config.defaults.agentId || undefined,
    appId: config.defaults.appId || undefined,
    runId: config.defaults.runId || undefined,
  };
}
