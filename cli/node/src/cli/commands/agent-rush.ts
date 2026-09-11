import { AgentRushError, requestAgentRush } from '../../backend/agent-rush.js';
import type { JsonObject } from '../../backend/json.js';
import { formatJsonEnvelope } from '../../output/format.js';
import { exit } from '../../runtime/exit.js';
import { isAgentMode } from '../../runtime/state.js';
/** AGENTRUSH 添加和搜索命令，项目由服务端路由。 */

import readline from 'node:readline';
import { loadConfig, saveConfig } from '../../config/store.js';
import { colors, printError, printSuccess } from '../../output/branding.js';

const PII_WARNING = [
  '',
  '⚠️  AGENTRUSH memories are PUBLIC — visible to any other player.',
  '   Do not include real names, emails, secrets, work content, or PII.',
  '',
].join('\n');

const ERROR_HINTS: Record<string, string> = {
  agentrush_search_first: "Run 3 'memgo agent-rush search' commands before adding.",
  agentrush_search_quota: "You've used your 3 lifetime searches.",
  agentrush_add_quota: "You've used your 3 lifetime adds.",
  agentrush_not_agent_mode: "Re-run 'memgo init --agent' to bootstrap an agent-mode key.",
  agentrush_length: 'Memory text must be 50-1000 characters.',
  agentrush_no_urls: 'URLs are not allowed.',
  agentrush_blocklist: 'Content contains a blocked term.',
  agentrush_global_quota: 'Event-wide cap reached. Try again later.',
  agentrush_not_provisioned: 'AGENTRUSH is not provisioned in this environment.',
};

/**
 * 读取凭据并调用 AGENTRUSH，失败时展示对应处理提示。
 * @param path - 业务接口路径。
 * @param body - 请求负载。
 * @returns 后端返回的业务对象。
 */
async function callEndpoint(path: string, body: JsonObject): Promise<JsonObject> {
  const config = loadConfig();

  if (!config.platform?.apiKey) {
    printError('Not initialized. Run `memgo init --agent` first.');
    exit(1);
  }

  try {
    return await requestAgentRush(config.platform, path, body);
  } catch (error) {
    if (!(error instanceof AgentRushError)) throw error;
    printError(error.message);
    if (ERROR_HINTS[error.code]) process.stderr.write(`  ${colors.dim(ERROR_HINTS[error.code])}\n`);
    exit(1);
  }
}

/**
 * 读取一行终端输入并清除首尾空白。
 * @param question - 终端提问文本。
 * @returns 清除首尾空白后的输入；支持默认值的入口在空输入时使用默认值。
 */
function promptLine(question: string): Promise<string> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

/** 交互环境首次确认记忆公开后记录时间；非交互环境只向 stderr 提示，保持无人值守流程。 */
async function ensureWarningAcknowledged(): Promise<void> {
  const config = loadConfig();
  if (config.agentRush?.acknowledgedAt) return;

  if (!process.stdin.isTTY || !process.stdout.isTTY) {
    // 非交互调用仅输出告警，不等待输入。
    console.error(PII_WARNING);
    return;
  }

  console.log(PII_WARNING);
  const answer = (await promptLine('   Continue? [y/N]: ')).toLowerCase();
  if (answer !== 'y' && answer !== 'yes') {
    printError('Aborted.');
    exit(1);
  }

  config.agentRush.acknowledgedAt = new Date().toISOString();
  saveConfig(config);
}

/**
 * 确认共享提示后提交记忆并输出结果。
 * @param content - 待写入的内容。
 */
export async function cmdAgentRushAdd(content: string): Promise<void> {
  await ensureWarningAcknowledged();
  const result = await callEndpoint('/v1/agent-rush/memories/', { content });
  if (isAgentMode()) {
    formatJsonEnvelope({ command: 'agent-rush add', data: result });
    return;
  }
  printSuccess(`Memory submitted (event_id: ${(result as { event_id?: string }).event_id ?? '?'})`);
}

/**
 * 搜索共享记忆并输出前几条结果或 JSON 信封。
 * @param query - 记忆搜索文本。
 */
export async function cmdAgentRushSearch(query: string): Promise<void> {
  const result = (await callEndpoint('/v1/agent-rush/memories/search/', {
    query,
  })) as {
    results?: Array<{ memory?: string }>;
    memories?: Array<{ memory?: string }>;
  };

  const memories = result.results ?? result.memories ?? [];
  if (isAgentMode()) {
    formatJsonEnvelope({ command: 'agent-rush search', data: memories });
    return;
  }

  if (memories.length === 0) {
    console.log(colors.dim('(no results)'));
    return;
  }

  memories.slice(0, 5).forEach((m, i) => {
    console.log(`  ${i + 1}. ${m.memory ?? JSON.stringify(m)}`);
  });
}
