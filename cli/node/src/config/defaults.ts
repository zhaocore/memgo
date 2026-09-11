import type { MemGoConfig } from './types.js';
export const DEFAULT_BASE_URL = 'https://api.memgo.ai';
export const CONFIG_VERSION = 1;
/**
 * 创建独立的默认配置对象。
 * @returns 不与其他调用共享子对象的默认配置。
 */
export function createDefaultConfig(): MemGoConfig {
  return {
    version: CONFIG_VERSION,
    defaults: {
      userId: '',
      agentId: '',
      appId: '',
      runId: '',
    },
    platform: {
      apiKey: '',
      baseUrl: DEFAULT_BASE_URL,
      userEmail: '',
      agentMode: false,
      createdVia: '',
      agentCaller: '',
      claimedAt: '',
      defaultUserId: '',
    },
    telemetry: {
      anonymousId: '',
    },
    agentRush: {
      acknowledgedAt: '',
    },
  };
}
