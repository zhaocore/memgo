import type { PlatformConfig } from '../config/types.js';
import { CLI_VERSION } from '../version.js';
import { requestOnce } from './http.js';
import { type JsonObject, jsonObject } from './json.js';

/** 保留平台错误码，由命令层选择对应操作提示。 */
export class AgentRushError extends Error {
  /**
   * 构造外部接口错误并保留失败上下文。
   * @param code - AGENTRUSH 业务错误码。
   */
  constructor(readonly code: string) {
    super(`AGENTRUSH error: ${code}`);
    this.name = 'AgentRushError';
  }
}
/**
 * 发送 AGENTRUSH 请求并校验响应对象与业务错误。
 * @param config - 当前配置。
 * @param endpoint - 相对于服务地址的接口路径。
 * @param body - 请求负载。
 * @returns 校验后的业务响应对象。
 */
export async function requestAgentRush(
  config: PlatformConfig,
  endpoint: string,
  body: JsonObject
): Promise<JsonObject> {
  const response = await requestOnce({
    url: `${config.baseUrl.replace(/\/+$/, '')}${endpoint}`,
    path: endpoint,
    method: 'POST',
    headers: {
      Authorization: `Token ${config.apiKey}`,
      'Content-Type': 'application/json',
      'X-MemGo-Source': 'cli',
      'X-MemGo-Client-Language': 'node',
      'X-MemGo-Client-Version': CLI_VERSION,
      'X-MemGo-Mode': 'agent-rush',
    },
    body: JSON.stringify(body),
    timeoutMs: 30000,
  });
  const result = jsonObject(await response.json());
  if (!response.ok) {
    const error = result.error;
    const code =
      error && typeof error === 'object' && !Array.isArray(error) ? error.code : undefined;
    throw new AgentRushError(typeof code === 'string' ? code : `HTTP ${response.status}`);
  }
  return result;
}
