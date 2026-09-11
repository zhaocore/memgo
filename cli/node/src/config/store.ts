import { syncApiKey } from '../integrations/plugin-sync.js';
import { resolveConfig } from './parse.js';
import type { MemGoConfig } from './types.js';
/** 配置优先级：命令行参数、环境变量、配置文件、默认值。 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

export const CONFIG_DIR = path.join(os.homedir(), '.memgo');
export const CONFIG_FILE = path.join(CONFIG_DIR, 'config.json');
/**
 * 创建配置目录并限制新目录访问权限。
 * @returns 配置目录的绝对路径。
 */
export function ensureConfigDir(): string {
  fs.mkdirSync(CONFIG_DIR, { recursive: true, mode: 0o700 });
  return CONFIG_DIR;
}

/**
 * 读取唯一配置文件；错误仅报告路径，不回显原始凭据。
 * @returns 按环境变量优先级解析后的配置。
 */
export function loadConfig(): MemGoConfig {
  if (!fs.existsSync(CONFIG_FILE)) return resolveConfig({}, process.env);
  let data: unknown;
  try {
    data = JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf-8'));
  } catch (error) {
    throw new Error(
      `Cannot read config ${CONFIG_FILE}: ${error instanceof SyntaxError ? 'invalid JSON' : 'file read failed'}`,
      { cause: error }
    );
  }
  return resolveConfig(data, process.env);
}

/**
 * 按持久化格式保存配置，并同步已有的插件密钥条目。
 * @param config - 当前配置。
 */
export function saveConfig(config: MemGoConfig): void {
  ensureConfigDir();

  const data = {
    version: config.version,
    defaults: {
      user_id: config.defaults.userId,
      agent_id: config.defaults.agentId,
      app_id: config.defaults.appId,
      run_id: config.defaults.runId,
    },
    platform: {
      api_key: config.platform.apiKey,
      base_url: config.platform.baseUrl,
      user_email: config.platform.userEmail,
      agent_mode: config.platform.agentMode,
      created_via: config.platform.createdVia,
      agent_caller: config.platform.agentCaller,
      claimed_at: config.platform.claimedAt,
      default_user_id: config.platform.defaultUserId,
    },
    telemetry: {
      anonymous_id: config.telemetry.anonymousId,
    },
    agent_rush: {
      acknowledged_at: config.agentRush.acknowledgedAt,
    },
  };

  fs.writeFileSync(CONFIG_FILE, JSON.stringify(data, null, 2));
  fs.chmodSync(CONFIG_FILE, 0o600);

  // 只同步已存在的插件或 shell 配置项；主配置保存后同步失败仅告警。

  if (config.platform.apiKey) {
    try {
      syncApiKey(config.platform.apiKey);
    } catch {
      process.stderr.write(
        `${JSON.stringify({ level: 'warn', event: 'plugin_sync_failed', message: 'Configuration saved; check plugin file permissions and JSON format' })}\n`
      );
    }
  }
}

export {
  CONFIG_VERSION,
  createDefaultConfig,
  DEFAULT_BASE_URL,
} from './defaults.js';
export type { MemGoConfig, PlatformConfig } from './types.js';

export { getNestedValue, redactKey, setNestedValue } from './resolve.js';
