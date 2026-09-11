import fs from 'node:fs';
import { identifyAgent, pingApiKey } from '../../backend/auth.js';
import { CONFIG_FILE, loadConfig, saveConfig } from '../../config/store.js';
import { warnOptional } from '../../runtime/warning.js';

/**
 * 探测密钥是否仍可复用；网络故障保留现有账号并告警。
 * @param apiKey - 访问服务的 API 密钥。
 * @param baseUrl - 服务基础地址。
 * @param timeoutMs - 单次请求超时，单位为毫秒。
 * @returns 密钥是否可继续复用；网络故障也返回真。
 */
export async function pingKey(
  apiKey: string,
  baseUrl: string,
  timeoutMs: number
): Promise<boolean> {
  // 仅 HTTP 401/403 表示密钥确定无效。
  // 网络故障不能视为无效密钥，避免重新创建账号并覆盖已有配置。

  try {
    const resp = await pingApiKey(baseUrl, apiKey, timeoutMs);
    return resp.status !== 401 && resp.status !== 403;
  } catch {
    warnOptional('key_validation_network');
    return true; // 状态未确认，不创建替代账号。
  }
}

/**
 * 存在调用方名称时同步平台身份及本地配置。
 * @param key - 访问服务的 API 密钥。
 * @param baseUrl - 服务基础地址。
 * @param agentCaller - 代理调用方名称。
 */
export async function maybeIdentify(
  key: string,
  baseUrl: string,
  agentCaller: string | undefined
): Promise<void> {
  // 复用密钥时同步显式声明的 agent_caller。

  if (!agentCaller) return;
  try {
    const resp = await identifyAgent(baseUrl, key, agentCaller);
    if (resp.ok) {
      try {
        const body = (await resp.json()) as { agent_caller?: string };
        if (fs.existsSync(CONFIG_FILE)) {
          const cfg = loadConfig();
          cfg.platform.agentCaller = body.agent_caller ?? agentCaller;
          saveConfig(cfg);
        }
      } catch {
        warnOptional('key');
      }
    }
  } catch {
    warnOptional('key');
  }
}
