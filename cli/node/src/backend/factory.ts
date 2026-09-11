import type { MemGoConfig } from '../config/types.js';
import { OSSBackend } from './oss.js';
import { PlatformBackend, type PlatformContext } from './platform.js';
import type { Backend } from './types.js';
/**
 * OSS 判定对齐 Go CLI factory.go (doc-01 §4.3):
 * base_url host 是平台域名 (api.memgo.ai) 或为空 → platform, 否则 → OSS。
 * @param baseURL - 服务基础地址。
 * @returns 地址是否属于托管平台。
 */
function isPlatformURL(baseURL: string): boolean {
  if (!baseURL) return true;
  const url = new URL(baseURL);
  if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password)
    throw new Error('API base URL must use HTTP(S) without embedded credentials');
  return url.hostname === 'api.memgo.ai';
}

/**
 * 按服务地址装配 Platform 或 OSS 连接器。
 * @param config - 当前配置。
 * @param context - 调用方提供的运行依赖。
 * @returns 与服务地址匹配的后端连接器。
 */
export function getBackend(config: MemGoConfig, context: PlatformContext): Backend {
  if (isPlatformURL(config.platform.baseUrl)) {
    return new PlatformBackend(config.platform, context);
  }
  return new OSSBackend(config.platform);
}
