import { PlatformBackend } from '../../backend/platform.js';
import type { MemGoConfig } from '../../config/store.js';
import { colors, printError, printInfo, printSuccess } from '../../output/branding.js';
import { exit } from '../../runtime/exit.js';
import { promptLine, promptSecret } from '../../runtime/prompt.js';
import { captureNotice, isAgentMode } from '../../runtime/state.js';
const { brand, dim } = colors;
/**
 * 交互读取密钥并构造新的平台配置。
 * @param config - 当前配置。
 * @returns 包含新密钥的平台配置。
 */
export async function setupPlatform(config: MemGoConfig): Promise<MemGoConfig> {
  console.log();
  console.log(
    `  ${dim('Get your API key at https://app.memgo.ai/dashboard/api-keys?utm_source=oss&utm_medium=cli-node')}`
  );
  console.log();

  const apiKey = await promptSecret(`  ${brand('API Key')}: `);
  if (!apiKey) {
    printError('API key is required.');
    exit(1);
  }
  return {
    ...config,
    platform: { ...config.platform, apiKey, createdVia: 'api_key' },
  };
}

/**
 * 交互读取默认用户 ID 并返回新配置。
 * @param config - 当前配置。
 * @returns 包含默认用户 ID 的新配置。
 */
export async function setupDefaults(config: MemGoConfig): Promise<MemGoConfig> {
  console.log();
  printInfo('Set default entity IDs (press Enter to skip).\n');

  const _systemUser = process.env.USER || process.env.USERNAME || 'memgo-cli';
  const userId = await promptLine(
    `  ${brand('Default User ID')} ${dim('(recommended)')}`,
    _systemUser
  );
  return {
    ...config,
    defaults: { ...config.defaults, userId: userId || config.defaults.userId },
  };
}

/**
 * 验证连接成功后返回邮箱更新，失败不保存成功配置。
 * @param config - 当前配置。
 * @returns 更新账号邮箱后的配置。
 */
export async function validatePlatform(config: MemGoConfig): Promise<MemGoConfig> {
  printInfo('Validating connection...');
  const backend = new PlatformBackend(config.platform, {
    callerType: () => (isAgentMode() ? 'agent' : 'user'),
    notice: captureNotice,
  });
  const ping = await backend.ping();
  printSuccess('Connected to memgo Platform!');
  return {
    ...config,
    platform: {
      ...config.platform,
      userEmail: typeof ping.user_email === 'string' ? ping.user_email : config.platform.userEmail,
    },
  };
}
