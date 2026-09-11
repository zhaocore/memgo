import type { JsonObject } from '../backend/json.js';
import type { MemGoConfig } from './types.js';
/**
 * 隐藏密钥中间内容，仅保留识别所需片段。
 * @param key - 访问服务的 API 密钥。
 * @returns 脱敏后的密钥展示文本。
 */
export function redactKey(key: string): string {
  if (!key) return '(not set)';
  if (key.length <= 8) return `${key.slice(0, 2)}***`;
  return `${key.slice(0, 4)}...${key.slice(-4)}`;
}

/** 配置路径与内存字段的对应表。 */
const KEY_MAP: Record<string, [keyof MemGoConfig, string]> = {
  'platform.api_key': ['platform', 'apiKey'],
  'platform.base_url': ['platform', 'baseUrl'],
  'platform.user_email': ['platform', 'userEmail'],
  'defaults.user_id': ['defaults', 'userId'],
  'defaults.agent_id': ['defaults', 'agentId'],
  'defaults.app_id': ['defaults', 'appId'],
  'defaults.run_id': ['defaults', 'runId'],
  // 短名称别名。
  api_key: ['platform', 'apiKey'],
  base_url: ['platform', 'baseUrl'],
  user_email: ['platform', 'userEmail'],
  user_id: ['defaults', 'userId'],
  agent_id: ['defaults', 'agentId'],
  app_id: ['defaults', 'appId'],
  run_id: ['defaults', 'runId'],
};

/**
 * 按外部点分键名读取配置字段。
 * @param config - 当前配置。
 * @param dottedKey - 持久化配置使用的点分字段名。
 * @returns 对应字段值；未知键返回未定义。
 */
export function getNestedValue(config: MemGoConfig, dottedKey: string): unknown {
  const mapping = KEY_MAP[dottedKey];
  if (!mapping) return undefined;
  const [section, field] = mapping;
  return (config[section] as unknown as JsonObject)[field];
}

/**
 * 返回新配置；未知字段明确报错，不修改输入。
 * @param config - 当前配置。
 * @param dottedKey - 持久化配置使用的点分字段名。
 * @param value - 待处理的字段值。
 * @returns 仅更新指定字段的新配置。
 */
export function setNestedValue(config: MemGoConfig, dottedKey: string, value: string): MemGoConfig {
  const mapping = KEY_MAP[dottedKey];
  if (!mapping) throw new Error(`Unknown config key: ${dottedKey}`);
  const [section, field] = mapping;
  return {
    ...config,
    [section]: { ...(config[section] as object), [field]: value },
  };
}
