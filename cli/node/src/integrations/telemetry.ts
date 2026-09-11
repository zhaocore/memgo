import type { JsonObject } from '../backend/json.js';
import { telemetrySenderPath } from '../runtime/assets.js';
/** 通过独立子进程发送 PostHog 遥测；MEMGO_TELEMETRY=false 时禁用。 */

import { spawn } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { CONFIG_FILE, loadConfig, saveConfig } from '../config/store.js';
import { CLI_VERSION } from '../version.js';

const POSTHOG_API_KEY = 'phc_hgJkUVJFYtmaJqrvf6CYN67TIQ8yhXAkWzUn9AMU4yX';
const POSTHOG_HOST = 'https://us.i.posthog.com/i/v0/e/';

/**
 * 读取环境变量中的遥测开关。
 * @returns 当前是否允许发送遥测。
 */
function isTelemetryEnabled(): boolean {
  return process.env.MEMGO_TELEMETRY !== 'false';
}

/**
 * 向标准错误输出结构化遥测失败告警。
 */
function warnTelemetry(): void {
  process.stderr.write(
    `${JSON.stringify({ level: 'warn', event: 'telemetry_failed', message: 'Optional telemetry could not be sent; primary command is unaffected' })}\n`
  );
}

/**
 * 持久化设备匿名标识，避免不同设备共用身份。
 * @returns 已有或新生成的匿名标识。
 */
function getOrCreateAnonymousId(): string {
  const config = loadConfig();
  if (config.telemetry.anonymousId) {
    return config.telemetry.anonymousId;
  }

  const newId = `cli-anon-${randomUUID().replace(/-/g, '')}`;
  config.telemetry.anonymousId = newId;
  try {
    saveConfig(config);
  } catch {
    warnTelemetry();
  }
  return newId;
}

/**
 * 身份优先级：缓存邮箱、密钥 MD5、设备匿名标识。
 * @returns 邮箱、密钥摘要或匿名标识。
 */
function getDistinctId(): string {
  try {
    const config = loadConfig();
    if (config.platform.userEmail) {
      return config.platform.userEmail;
    }
    if (config.platform.apiKey) {
      return createHash('md5').update(config.platform.apiKey).digest('hex');
    }
  } catch {
    warnTelemetry();
  }
  try {
    return getOrCreateAnonymousId();
  } catch {
    return `cli-anon-${randomUUID().replace(/-/g, '')}`;
  }
}

/**
 * 异步启动遥测发送器；已有邮箱时省去重复探活，故障仅告警。
 * @param eventName - 遥测事件名称。
 * @param properties - 事件附加属性。
 * @param preResolvedEmail - 已经取得的账号邮箱，避免重复解析身份。
 */
export function captureEvent(
  eventName: string,
  properties: JsonObject,
  preResolvedEmail?: string
): void {
  if (!isTelemetryEnabled()) return;

  try {
    const config = loadConfig();
    const distinctId = preResolvedEmail || getDistinctId();

    // 首次识别真实身份时发送 $identify，将匿名历史关联到该身份。

    // 清除已关联的匿名标识，避免重复关联。
    let anonIdToAlias: string | null = null;
    if (distinctId && !distinctId.startsWith('cli-anon-') && config.telemetry.anonymousId) {
      anonIdToAlias = config.telemetry.anonymousId;
      config.telemetry.anonymousId = '';
      try {
        saveConfig(config);
      } catch {
        warnTelemetry();
      }
    }

    // 每个 cli.* 事件包含配置中的 agent_mode，串联初始化、添加和搜索流程。

    const payload = {
      api_key: POSTHOG_API_KEY,
      distinct_id: distinctId,
      event: eventName,
      properties: {
        source: 'CLI',
        language: 'node',
        cli_version: CLI_VERSION,
        agent_mode: Boolean(config.platform.agentMode),
        node_version: process.version,
        os: process.platform,
        ...properties,
        $process_person_profile: false,
        $lib: 'posthog-node',
      },
    };

    const context = {
      payload,
      posthogHost: POSTHOG_HOST,
      needsEmail: !distinctId || !distinctId.includes('@'),
      memgoApiKey: config.platform.apiKey || '',
      memgoBaseUrl: config.platform.baseUrl || 'https://api.memgo.ai',
      configPath: CONFIG_FILE,
      anonDistinctIdToAlias: anonIdToAlias,
    };

    const child = spawn(process.execPath, [telemetrySenderPath()], {
      detached: true,
      stdio: ['pipe', 'ignore', 'ignore'],
    });
    child.on('error', warnTelemetry);
    child.stdin?.on('error', warnTelemetry);
    child.stdin?.end(JSON.stringify(context));
    child.unref();
  } catch {
    warnTelemetry();
  }
}
