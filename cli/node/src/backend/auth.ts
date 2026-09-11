import { requestOnce } from './http.js';
import type { JsonObject } from './json.js';

const sourceHeaders = {
  'Content-Type': 'application/json',
  'X-MemGo-Source': 'cli',
  'X-MemGo-Client-Language': 'node',
};
/**
 * 认证写操作不自动重试，避免重复注册或重复发送验证码。
 * @param baseUrl - 服务基础地址。
 * @param endpoint - 相对于服务地址的接口路径。
 * @param method - HTTP 方法。
 * @param body - 请求负载。
 * @param key - 访问服务的 API 密钥。
 * @returns HTTP 响应，状态由调用方进一步检查。
 */
function send(
  baseUrl: string,
  endpoint: string,
  method: string,
  body: JsonObject,
  key: string | undefined
): Promise<Response> {
  return requestOnce({
    url: `${baseUrl.replace(/\/+$/, '')}${endpoint}`,
    path: endpoint,
    method,
    headers: key ? { ...sourceHeaders, Authorization: `Token ${key}` } : sourceHeaders,
    body: JSON.stringify(body),
    timeoutMs: 30000,
  });
}
/**
 * 请求向指定邮箱发送验证码。
 * @param baseUrl - 服务基础地址。
 * @param email - 用于登录或认领的邮箱。
 * @returns 发送验证码请求的 HTTP 响应。
 */
export function sendEmailCode(baseUrl: string, email: string): Promise<Response> {
  return send(baseUrl, '/api/v1/auth/email_code/', 'POST', { email }, undefined);
}
/**
 * 验证邮箱验证码，并在提供代理密钥时认领账号。
 * @param baseUrl - 服务基础地址。
 * @param email - 用于登录或认领的邮箱。
 * @param code - 邮箱验证码。
 * @param agentKey - 需要认领的代理账号密钥。
 * @returns 验证码校验请求的 HTTP 响应。
 */
export function verifyEmailCode(
  baseUrl: string,
  email: string,
  code: string,
  agentKey: string | undefined
): Promise<Response> {
  return send(
    baseUrl,
    '/api/v1/auth/email_code/verify/',
    'POST',
    { email, code, ...(agentKey ? { agent_mode_api_key: agentKey } : {}) },
    undefined
  );
}
/**
 * 请求创建未认领的代理账号。
 * @param baseUrl - 服务基础地址。
 * @param body - 请求负载。
 * @returns 创建账号请求的 HTTP 响应。
 */
export function bootstrapAgent(baseUrl: string, body: JsonObject): Promise<Response> {
  return send(baseUrl, '/api/v1/auth/agent_mode/', 'POST', body, undefined);
}
/**
 * 更新平台记录的代理调用方名称。
 * @param baseUrl - 服务基础地址。
 * @param key - 访问服务的 API 密钥。
 * @param name - 调用方或命令名称。
 * @returns 更新调用方请求的 HTTP 响应。
 */
export function identifyAgent(baseUrl: string, key: string, name: string): Promise<Response> {
  return send(baseUrl, '/api/v1/auth/agent_mode/caller/', 'PATCH', { agent_caller: name }, key);
}
/**
 * 在指定超时内发送密钥探活请求。
 * @param baseUrl - 服务基础地址。
 * @param key - 访问服务的 API 密钥。
 * @param timeoutMs - 单次请求超时，单位为毫秒。
 * @returns 探活请求的 HTTP 响应。
 */
export function pingApiKey(baseUrl: string, key: string, timeoutMs: number): Promise<Response> {
  return requestOnce({
    url: `${baseUrl.replace(/\/+$/, '')}/v1/ping/`,
    path: '/v1/ping/',
    method: 'GET',
    headers: { Authorization: `Token ${key}` },
    body: undefined,
    timeoutMs,
  });
}
