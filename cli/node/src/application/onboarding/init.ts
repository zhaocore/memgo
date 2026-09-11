import fs from 'node:fs';
import readline from 'node:readline';
import type { JsonObject } from '../../backend/json.js';
import {
  CONFIG_FILE,
  DEFAULT_BASE_URL,
  createDefaultConfig,
  loadConfig,
  redactKey,
  saveConfig,
} from '../../config/store.js';
import { colors, printBanner, printError, printInfo, printSuccess } from '../../output/branding.js';
import { formatJsonEnvelope } from '../../output/format.js';
import { exit } from '../../runtime/exit.js';
import { promptLine } from '../../runtime/prompt.js';
import { emailLogin, validateEmail } from './email.js';
import { maybeIdentify, pingKey } from './key.js';
import { setupDefaults, setupPlatform, validatePlatform } from './setup.js';
const { brand, dim } = colors;
/**
 * 按显式参数与运行环境完成初始化、账号认领或密钥复用。
 * @param input - 调用方提供的输入。
 * @param input.apiKey - 访问服务的 API 密钥。
 * @param input.userId - 用户 ID。
 * @param input.email - 用于登录或认领的邮箱。
 * @param input.code - 邮箱验证码。
 * @param input.force - 跳过已有配置或删除操作的交互确认。
 * @param input.agent - 显式请求代理初始化流程。
 * @param input.source - 账号创建来源。
 * @param input.agentCaller - 代理调用方名称。
 */
export async function runInit(input: {
  apiKey?: string;
  userId?: string;
  email?: string;
  code?: string;
  force?: boolean;
  agent?: boolean;
  source?: string;
  agentCaller?: string;
}): Promise<void> {
  const opts = { ...input };
  const { detectAgentCaller } = await import('../../integrations/agent-detect.js');
  const { bootstrapViaBackend, claimViaOtp } = await import('./agent-mode.js');
  const { isAgentMode } = await import('../../runtime/state.js');
  const { captureEvent } = await import('../../integrations/telemetry.js');

  const fireInit = (mode: 'agent' | 'email' | 'api_key' | 'existing_key', claimed: boolean) => {
    const props: JsonObject = { command: 'init', mode };
    // 代理身份由 --agent-caller 声明，不从环境推断。
    if (opts.agentCaller) props.agent_caller = opts.agentCaller;
    if (opts.source) props.signup_source = opts.source;
    if (claimed) props.claimed_agent_mode = true;
    captureEvent('cli.init', props);
  };

  let config = createDefaultConfig();
  const savedConfig = loadConfig();
  const baseUrl = process.env.MEMGO_BASE_URL || savedConfig.platform.baseUrl || DEFAULT_BASE_URL;
  config.platform.baseUrl = baseUrl;

  // 校验互斥参数。
  if (opts.code && !opts.email) {
    printError('--code requires --email.');
    exit(1);
  }
  if (opts.email && opts.apiKey) {
    printError('Cannot use both --api-key and --email.');
    exit(1);
  }

  // 邮箱认领已有 Agent Mode 账号。
  if (
    opts.email &&
    fs.existsSync(CONFIG_FILE) &&
    savedConfig.platform.agentMode &&
    savedConfig.platform.apiKey
  ) {
    const email = opts.email.trim().toLowerCase();
    validateEmail(email);
    printInfo(`Claiming Agent Mode account to ${email}...`);
    await claimViaOtp(savedConfig, { email, code: opts.code });
    fireInit('email', true);
    return;
  }

  // Agent Mode 分支先于覆盖配置确认执行。
  // 已有有效密钥直接复用，不询问是否覆盖。

  // 仅在没有可复用密钥时创建账号。

  const agentCtx = opts.agent === true || isAgentMode() || detectAgentCaller() !== null;
  if (!opts.apiKey && !opts.email && agentCtx) {
    const emitReuseEnvelope = (source: 'env' | 'config') => {
      if (isAgentMode()) {
        formatJsonEnvelope({
          command: 'init',
          data: {
            api_key_saved: false,
            api_key_source: source,
            agent_mode: false,
            message: 'Existing MemGo API key found and reused. No Agent Mode key was created.',
          },
        });
      } else {
        printSuccess(
          source === 'env'
            ? 'Existing MEMGO_API_KEY is valid; reusing it. No new Agent Mode key was minted.'
            : 'Existing API key in config is valid; reusing it. No new Agent Mode key was minted.'
        );
      }
    };
    // 优先复用环境变量中的有效密钥。
    const envKey = (process.env.MEMGO_API_KEY || '').trim();
    if (envKey && (await pingKey(envKey, baseUrl, 5000))) {
      await maybeIdentify(envKey, baseUrl, opts.agentCaller);
      emitReuseEnvelope('env');
      fireInit('existing_key', false);
      return;
    }
    // 其次复用配置文件中的有效密钥。
    if (
      savedConfig.platform.apiKey &&
      (await pingKey(savedConfig.platform.apiKey, baseUrl, 5000))
    ) {
      await maybeIdentify(savedConfig.platform.apiKey, baseUrl, opts.agentCaller);
      emitReuseEnvelope('config');
      fireInit('existing_key', false);
      return;
    }
    // 没有有效密钥时才创建未认领账号。
    // 身份由 --agent-caller 提供；环境检测只用于判断是否进入代理流程。

    await bootstrapViaBackend(config, {
      source: opts.source ?? null,
      agentCaller: opts.agentCaller ?? null,
    });
    fireInit('agent', false);
    return;
  }

  // 覆盖已有密钥配置前提示。
  if (!opts.force && fs.existsSync(CONFIG_FILE) && savedConfig.platform.apiKey) {
    console.log(
      `\n  ${brand('Existing configuration found')} ${dim(`(API key: ${redactKey(savedConfig.platform.apiKey)})`)}`
    );
    if (process.stdin.isTTY) {
      const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
      });
      const answer = await new Promise<string>((resolve) => {
        rl.question('  Overwrite existing config? This cannot be undone. [y/N] ', resolve);
      });
      rl.close();
      if (answer.toLowerCase() !== 'y') {
        printInfo('Cancelled. Use --force to skip this check.');
        exit(0);
      }
    } else {
      printError('Existing config would be overwritten.', 'Use --force to overwrite.');
      exit(1);
    }
  }

  // 邮箱登录流程。
  if (opts.email) {
    const email = opts.email.trim().toLowerCase();
    validateEmail(email);

    printBanner();
    console.log();
    printInfo(`Logging in as ${email}...\n`);

    const result = await emailLogin(email, opts.code, baseUrl);

    const apiKeyVal = result.api_key as string | undefined;
    if (!apiKeyVal) {
      printError('Auth succeeded but no API key was returned. Contact support.');
      exit(1);
    }

    config.platform.apiKey = apiKeyVal;
    config.platform.baseUrl = baseUrl;
    config.platform.userEmail = email;
    config.platform.createdVia = 'email';
    config.defaults.userId = opts.userId || process.env.USER || process.env.USERNAME || 'memgo-cli';

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
      return;
    }
    console.log();
    printSuccess('Authenticated! Configuration saved to ~/.memgo/config.json');
    console.log();
    console.log(`  ${dim('Get started:')}`);
    console.log(`  ${dim('  memgo add "I prefer dark mode"')}`);
    console.log(`  ${dim('  memgo search "preferences"')}`);
    console.log();
    return;
  }

  // API key 配置流程。
  // Agent Mode 已在覆盖确认之前处理。

  // 非交互环境从已有默认值补齐参数。
  if (!process.stdin.isTTY) {
    if (!opts.apiKey) {
      printError(
        'Non-interactive terminal detected and --api-key is required.',
        'Usage: memgo init --api-key <key>, --email <addr>, or --agent for unattended Agent Mode bootstrap.'
      );
      exit(1);
    }
    opts.userId = opts.userId || process.env.USER || process.env.USERNAME || 'memgo-cli';
  }

  // 非交互参数已齐全。
  if (opts.apiKey && opts.userId) {
    config.platform.apiKey = opts.apiKey;
    config.platform.createdVia = 'api_key';
    config.defaults.userId = opts.userId;
    config = await validatePlatform(config);
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
      return;
    }
    printSuccess('Configuration saved to ~/.memgo/config.json');
    return;
  }

  printBanner();
  console.log();
  printInfo("Welcome! Let's set up your memgo CLI.\n");

  // 使用传入密钥，缺失时交互读取。
  if (opts.apiKey) {
    config.platform.apiKey = opts.apiKey;
  } else {
    console.log(`  ${brand('How would you like to authenticate?')}`);
    console.log(`  ${dim('1.')} Login with email ${dim('(recommended)')}`);
    console.log(`  ${dim('2.')} Enter API key manually`);
    console.log();

    const choice = await promptLine(`  ${brand('Choose')} [1/2]`, '1');

    if (choice === '1') {
      console.log();
      const emailAddr = await promptLine(`  ${brand('Email')}`);
      if (!emailAddr) {
        printError('Email is required.');
        exit(1);
      }

      const email = emailAddr.trim().toLowerCase();
      validateEmail(email);
      printInfo(`Logging in as ${email}...\n`);

      const result = await emailLogin(email, undefined, baseUrl);

      const apiKeyVal = result.api_key as string | undefined;
      if (!apiKeyVal) {
        printError('Auth succeeded but no API key was returned. Contact support.');
        exit(1);
      }

      config.platform.apiKey = apiKeyVal;
      config.platform.baseUrl = baseUrl;
      config.platform.userEmail = email;
      config.platform.createdVia = 'email';
      config.defaults.userId =
        opts.userId || process.env.USER || process.env.USERNAME || 'memgo-cli';

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
        return;
      }
      console.log();
      printSuccess('Authenticated! Configuration saved to ~/.memgo/config.json');
      console.log();
      console.log(`  ${dim('Get started:')}`);
      console.log(`  ${dim('  memgo add "I prefer dark mode"')}`);
      console.log(`  ${dim('  memgo search "preferences"')}`);
      console.log();
      return;
    }

    // 选择密钥方式后继续读取。
    config = await setupPlatform(config);
  }

  // 使用传入用户标识，缺失时交互读取。
  if (opts.userId) {
    config.defaults.userId = opts.userId;
  } else {
    config = await setupDefaults(config);
  }

  config = await validatePlatform(config);

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
    return;
  }
  console.log();
  printSuccess('Configuration saved to ~/.memgo/config.json');
  console.log();
  console.log(`  ${dim('Get started:')}`);
  if (config.defaults.userId) {
    console.log(`  ${dim('  memgo add "I prefer dark mode"')}`);
    console.log(`  ${dim('  memgo search "preferences"')}`);
  } else {
    console.log(`  ${dim('  memgo add "I prefer dark mode" --user-id alice')}`);
    console.log(`  ${dim('  memgo search "preferences" --user-id alice')}`);
  }
  console.log();
}
