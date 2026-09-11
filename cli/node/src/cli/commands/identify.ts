import { identifyAgent } from '../../backend/auth.js';
import { formatJsonEnvelope } from '../../output/format.js';
import { exit } from '../../runtime/exit.js';
import { isAgentMode } from '../../runtime/state.js';
/** 通过 identify 声明当前代理身份；可为初始化时未声明身份的密钥补齐信息。 */

import { loadConfig, saveConfig } from '../../config/store.js';
import { printError, printSuccess } from '../../output/branding.js';

/**
 * 更新代理调用方名称并同步本地配置。
 * @param name - 调用方或命令名称。
 */
export async function runIdentify(name: string): Promise<void> {
  const config = loadConfig();
  if (!config.platform.apiKey) {
    printError('No API key configured. Run `memgo init --agent` first.');
    exit(1);
  }
  if (!config.platform.agentMode) {
    printError('This command only works on unclaimed agent-mode keys.');
    exit(1);
  }

  const clean = (name ?? '').trim();
  if (!clean) {
    printError('Agent name is required.');
    exit(1);
  }

  const baseUrl = (config.platform.baseUrl || 'https://api.memgo.ai').replace(/\/+$/, '');

  let resp: Response;
  try {
    resp = await identifyAgent(baseUrl, config.platform.apiKey, clean);
  } catch (err) {
    printError(`Network error: ${err instanceof Error ? err.message : String(err)}`);
    exit(1);
  }

  if (!resp.ok) {
    let detail: string = resp.statusText;
    try {
      const body = (await resp.json()) as { error?: string };
      if (body.error) detail = body.error;
    } catch {
      /* 非 JSON 错误响应保留 HTTP 状态说明。 */
    }
    printError(`Identify failed: ${detail}`);
    exit(1);
  }

  const body = (await resp.json()) as { agent_caller?: string };
  const canonical = body.agent_caller ?? clean;
  config.platform.agentCaller = canonical;
  saveConfig(config);
  if (isAgentMode()) {
    formatJsonEnvelope({
      command: 'identify',
      data: { agent_caller: canonical },
    });
    return;
  }
  printSuccess(`Identified as ${canonical}.`);
}
