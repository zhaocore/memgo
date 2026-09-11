import { CONFIG_VERSION, DEFAULT_BASE_URL } from './defaults.js';
import type { MemGoConfig } from './types.js';

/**
 * 校验配置对象；空缺字段返回空对象供默认值解析。
 * @param value - 待处理的字段值。
 * @param field - 用于错误定位的配置字段路径。
 * @returns 通过校验的对象，空缺输入对应空对象。
 */
function object(value: unknown, field: string): Record<string, unknown> {
  if (value === null || value === undefined) return {};
  if (typeof value !== 'object' || Array.isArray(value))
    throw new Error(`Invalid config field ${field}: expected object`);
  return value as Record<string, unknown>;
}
/**
 * 校验配置字符串；空缺字段使用调用方指定值。
 * @param value - 待处理的字段值。
 * @param field - 用于错误定位的配置字段路径。
 * @param absent - 字段缺失时使用的值。
 * @returns 配置字符串或显式提供的缺省值。
 */
function string(value: unknown, field: string, absent: string): string {
  if (value === undefined || value === null) return absent;
  if (typeof value !== 'string') throw new Error(`Invalid config field ${field}: expected string`);
  return value;
}
/**
 * 校验持久化输入并返回新模型；环境变量覆盖文件，空变量保持既有语义。
 * @param raw - 从配置文件读取的原始数据。
 * @param env - 调用环境中的变量。
 * @returns 合并环境变量和文件字段后的新配置。
 */
export function resolveConfig(raw: unknown, env: NodeJS.ProcessEnv): MemGoConfig {
  const data = object(raw, 'root');
  const platform = object(data.platform, 'platform');
  const defaults = object(data.defaults, 'defaults');
  const telemetry = object(data.telemetry, 'telemetry');
  const rush = object(data.agent_rush, 'agent_rush');
  const version = data.version ?? CONFIG_VERSION;
  if (typeof version !== 'number') throw new Error('Invalid config field version: expected number');
  const agentMode = platform.agent_mode ?? false;
  if (typeof agentMode !== 'boolean')
    throw new Error('Invalid config field platform.agent_mode: expected boolean');
  return {
    version,
    platform: {
      apiKey: env.MEMGO_API_KEY || string(platform.api_key, 'platform.api_key', ''),
      baseUrl:
        env.MEMGO_BASE_URL || string(platform.base_url, 'platform.base_url', DEFAULT_BASE_URL),
      userEmail: string(platform.user_email, 'platform.user_email', ''),
      agentMode,
      createdVia: string(platform.created_via, 'platform.created_via', ''),
      agentCaller: string(platform.agent_caller, 'platform.agent_caller', ''),
      claimedAt: string(platform.claimed_at, 'platform.claimed_at', ''),
      defaultUserId: string(platform.default_user_id, 'platform.default_user_id', ''),
    },
    defaults: {
      userId: env.MEMGO_USER_ID || string(defaults.user_id, 'defaults.user_id', ''),
      agentId: env.MEMGO_AGENT_ID || string(defaults.agent_id, 'defaults.agent_id', ''),
      appId: env.MEMGO_APP_ID || string(defaults.app_id, 'defaults.app_id', ''),
      runId: env.MEMGO_RUN_ID || string(defaults.run_id, 'defaults.run_id', ''),
    },
    telemetry: {
      anonymousId: string(telemetry.anonymous_id, 'telemetry.anonymous_id', ''),
    },
    agentRush: {
      acknowledgedAt: string(rush.acknowledged_at, 'agent_rush.acknowledged_at', ''),
    },
  };
}
