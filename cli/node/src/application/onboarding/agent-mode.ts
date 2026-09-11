import { bootstrapAgent, sendEmailCode, verifyEmailCode } from '../../backend/auth.js';
import type { JsonObject } from '../../backend/json.js';
import { formatJsonEnvelope } from '../../output/format.js';
import { exit } from '../../runtime/exit.js';
import { isAgentMode } from '../../runtime/state.js';
/** Agent Mode 初始化与验证码认领流程。 */

import readline from 'node:readline';
import { type MemGoConfig, saveConfig } from '../../config/store.js';
import { colors, printError, printSuccess } from '../../output/branding.js';

const { brand, dim } = colors;

export interface BootstrapEnvelope {
  api_key: string;
  default_user_id: string;
  claim_command?: string;
  memgo_notice?: string;
}

/**
 * 检查代理账号响应中密钥与默认用户 ID 是否有效。
 * @param v - 待校验的接口响应。
 * @returns 响应是否包含有效密钥与默认用户 ID。
 */
function isValidEnvelope(v: unknown): v is BootstrapEnvelope {
  return (
    !!v &&
    typeof v === 'object' &&
    typeof (v as BootstrapEnvelope).api_key === 'string' &&
    (v as BootstrapEnvelope).api_key.length > 0 &&
    typeof (v as BootstrapEnvelope).default_user_id === 'string' &&
    (v as BootstrapEnvelope).default_user_id.length > 0
  );
}

/**
 * 调用初始化接口，保存返回的配置。source 为渠道，agentCaller 是调用方显式声明的代理身份。
 * @param input - 账号操作前的配置。
 * @param root0 - 账号操作选项。
 * @param root0.source - 账号创建来源。
 * @param root0.agentCaller - 代理调用方名称。
 * @returns 已保存代理账号信息的新配置。
 */
export async function bootstrapViaBackend(
  input: MemGoConfig,
  { source, agentCaller }: { source?: string | null; agentCaller?: string | null }
): Promise<MemGoConfig> {
  const config: MemGoConfig = {
    ...input,
    platform: { ...input.platform },
    defaults: { ...input.defaults },
  };
  const baseUrl = (config.platform.baseUrl || 'https://api.memgo.ai').replace(/\/+$/, '');
  const body: JsonObject = {};
  if (source) body.source = source;
  if (agentCaller) body.agent_caller = agentCaller;

  let resp: Response;
  try {
    resp = await bootstrapAgent(baseUrl, body);
  } catch (err) {
    printError(
      `Network error contacting MemGo: ${err instanceof Error ? err.message : String(err)}`
    );
    exit(1);
  }

  if (resp.status === 429) {
    printError('Rate-limited. Try again in a few minutes.');
    exit(1);
  }
  if (resp.status === 503) {
    printError('Agent Mode is temporarily disabled. Try again later.');
    exit(1);
  }
  if (!resp.ok) {
    let detail: string = resp.statusText;
    try {
      const errBody = (await resp.json()) as {
        error?: string;
        detail?: string;
      };
      detail = errBody.error ?? errBody.detail ?? resp.statusText;
    } catch {
      /* 非 JSON 错误响应保留 HTTP 状态说明。 */
    }
    // 将平台返回的通用权限拒绝映射为现有注册限流提示。

    if (resp.status === 403 && /permission/i.test(detail)) {
      printError(
        'Daily Agent Mode signup limit reached for this network (5/day). Try again from a different IP or after midnight UTC.'
      );
      exit(1);
    }
    printError(`Bootstrap failed: ${detail}`);
    exit(1);
  }

  const envelope = (await resp.json()) as BootstrapEnvelope;
  if (!isValidEnvelope(envelope)) {
    // 校验初始化响应必需字段，避免将空密钥写入配置。

    printError('Bootstrap response missing required fields — please update the CLI.');
    exit(1);
  }

  config.platform.apiKey = envelope.api_key;
  config.platform.baseUrl = baseUrl;
  config.platform.agentMode = true;
  config.platform.createdVia = 'agent_mode';
  config.platform.agentCaller = agentCaller ?? '';
  config.platform.claimedAt = '';
  config.platform.defaultUserId = envelope.default_user_id;
  // 以初始化返回的用户标识作为记忆操作默认范围。
  config.defaults.userId = envelope.default_user_id;
  saveConfig(config);
  if (isAgentMode()) {
    formatJsonEnvelope({
      command: 'init',
      data: {
        api_key_saved: true,
        agent_mode: config.platform.agentMode,
        user_id: config.defaults.userId,
      },
    });
    return config;
  }

  printSuccess(`Agent Mode active. Default user_id: ${envelope.default_user_id}`);
  if (envelope.memgo_notice) {
    console.log(`\n\x1b[33m🔔 ${envelope.memgo_notice}\x1b[0m\n`);
  } else {
    // 旧服务没有统一提示字段时保留现有认领说明。
    const claimCmd = envelope.claim_command ?? 'memgo init --email <your-email>';
    console.log(`  ${dim(`To claim this account later: ${claimCmd}`)}`);
  }

  if (!agentCaller) {
    console.log(
      `  ${dim('If you are an AI agent, identify yourself: `memgo identify <your-name>` (e.g. claude-code, cursor).')}`
    );
  }
  return config;
}

/**
 * 通过邮箱验证码原地认领账号；携带 agent_mode_api_key 验证，不创建替代账号。
 * @param input - 账号操作前的配置。
 * @param root0 - 账号操作选项。
 * @param root0.email - 用于登录或认领的邮箱。
 * @param root0.code - 邮箱验证码。
 * @returns 已认领账号的新配置。
 */
export async function claimViaOtp(
  input: MemGoConfig,
  { email, code }: { email: string; code?: string }
): Promise<MemGoConfig> {
  const config: MemGoConfig = {
    ...input,
    platform: { ...input.platform },
    defaults: { ...input.defaults },
  };
  const baseUrl = (config.platform.baseUrl || 'https://api.memgo.ai').replace(/\/+$/, '');
  if (!config.platform.apiKey || !config.platform.agentMode) {
    printError('This command requires an active Agent Mode config. Run `memgo init` first.');
    exit(1);
  }

  const rawKey = config.platform.apiKey;

  // 未提供验证码时先请求发送。
  if (!code) {
    const sendResp = await sendEmailCode(baseUrl, email);
    if (sendResp.status === 429) {
      printError('Too many attempts. Try again in a few minutes.');
      exit(1);
    }
    if (!sendResp.ok) {
      let detail: string = sendResp.statusText;
      try {
        const errBody = (await sendResp.json()) as { error?: string };
        if (errBody.error) detail = errBody.error;
      } catch {
        /* 非 JSON 错误响应保留 HTTP 状态说明。 */
      }
      printError(`Failed to send code: ${detail}`);
      exit(1);
    }

    printSuccess(`Verification code sent to ${email}. Check your inbox.`);

    if (!process.stdin.isTTY) {
      printError(
        'No --code provided and terminal is non-interactive.',
        `Re-run: memgo init --email ${email} --code <code>`
      );
      exit(1);
    }

    console.log();
    code = await promptLine(`  ${brand('Verification Code')}`);
    if (!code) {
      printError('Code is required.');
      exit(1);
    }
  }

  // 验证码验证与账号认领由服务端原子执行。
  const verifyResp = await verifyEmailCode(baseUrl, email, code.trim(), rawKey);

  if (!verifyResp.ok) {
    let detail: string = verifyResp.statusText;
    let errCode = '';
    try {
      const errBody = (await verifyResp.json()) as {
        error?: string;
        code?: string;
      };
      if (errBody.error) detail = errBody.error;
      if (errBody.code) errCode = errBody.code;
    } catch {
      /* 非 JSON 错误响应保留 HTTP 状态说明。 */
    }
    printError(`Claim failed: ${detail}`);
    if (errCode === 'email_already_claimed') {
      console.log(
        `  ${dim('Tip: this email already has a MemGo account. Sign in at app.memgo.ai with your existing credentials.')}`
      );
    }
    exit(1);
  }

  const claimBody = (await verifyResp.json()) as {
    claimed?: boolean;
    claimed_at?: string;
  };
  if (!claimBody.claimed) {
    printError('Unexpected verify response: claimed must be true');
    exit(1);
  }

  config.platform.agentMode = false;
  config.platform.claimedAt = claimBody.claimed_at ?? new Date().toISOString();
  config.platform.userEmail = email;
  config.platform.createdVia = 'email';
  saveConfig(config);
  if (isAgentMode()) {
    formatJsonEnvelope({
      command: 'init',
      data: {
        api_key_saved: true,
        agent_mode: config.platform.agentMode,
        user_id: config.defaults.userId,
      },
    });
    return config;
  }

  printSuccess(`Agent claimed to ${email}. Your API key is unchanged.`);
  return config;
}

/**
 * 读取一行终端输入并清除首尾空白。
 * @param label - 终端提示文本。
 * @returns 清除首尾空白后的输入；支持默认值的入口在空输入时使用默认值。
 */
function promptLine(label: string): Promise<string> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => {
    rl.question(`${label}: `, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}
