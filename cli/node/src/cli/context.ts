import { AuthError, type Backend, getBackend } from '../backend/index.js';
import { loadConfig, saveConfig } from '../config/store.js';
import type { MemGoConfig } from '../config/types.js';
import { printError } from '../output/branding.js';
import { exit } from '../runtime/exit.js';
import { captureNotice, isAgentMode } from '../runtime/state.js';

/**
 * 合并调用参数与配置，验证连接并更新账号邮箱。
 * @param apiKey - 访问服务的 API 密钥。
 * @param baseUrl - 服务基础地址。
 * @returns 通过预检的后端与有效配置。
 */
export async function getBackendAndConfig(
  apiKey?: string,
  baseUrl?: string
): Promise<{ backend: Backend; config: MemGoConfig }> {
  const saved = loadConfig();
  const config = {
    ...saved,
    platform: {
      ...saved.platform,
      apiKey: apiKey || saved.platform.apiKey,
      baseUrl: baseUrl || saved.platform.baseUrl,
    },
  };

  if (!config.platform.apiKey) {
    printError(
      'No API key configured.',
      "Run 'memgo init' or set MEMGO_API_KEY environment variable."
    );
    exit(1);
  }

  const backend = getBackend(config, {
    callerType: () => (isAgentMode() ? 'agent' : 'user'),
    notice: captureNotice,
  });

  try {
    const pingData = await backend.ping();

    const email = pingData?.user_email as string | undefined;
    if (email) {
      if (config.platform.userEmail !== email) {
        config.platform.userEmail = email;
        saveConfig(config);
      }
    }
  } catch (e) {
    if (e instanceof AuthError) {
      printError(
        'Invalid or expired API key.',
        "Run 'memgo init' or set MEMGO_API_KEY environment variable."
      );
      exit(1);
    }
    throw e;
  }

  return { backend, config };
}

/**
 * 完成配置与鉴权预检后取得后端连接器。
 * @param apiKey - 访问服务的 API 密钥。
 * @param baseUrl - 服务基础地址。
 * @returns 通过预检的后端连接器。
 */
export async function getBackendOnly(apiKey?: string, baseUrl?: string): Promise<Backend> {
  return (await getBackendAndConfig(apiKey, baseUrl)).backend;
}
