import { setTimeout as delay } from 'node:timers/promises';
import { APIError, AuthError, NotFoundError } from './errors.js';

/** 网络边界接收 JSON，解析后由协议适配器校验所需字段。 */
export interface HttpRequest {
  url: string;
  path: string;
  method: string;
  headers: Record<string, string>;
  body: string | undefined;
  timeoutMs: number;
}

/**
 * 清除响应中回显的凭据及敏感字段，避免错误消息泄露密钥。
 * @param value - 可能回显敏感字段的响应文本。
 * @param headers - 请求头，包含需要从响应中去除的凭据。
 * @returns 移除敏感字段与请求凭据后的文本。
 */
export function redactResponse(value: string, headers: Record<string, string>): string {
  let result = value;
  for (const [name, secret] of Object.entries(headers)) {
    if (/authorization|api[-_]key/i.test(name) && secret) {
      result = result.split(secret).join('[redacted]');
      const token = secret.replace(/^(Token|Bearer) /i, '');
      if (token) result = result.split(token).join('[redacted]');
    }
  }
  return result.replace(
    /("(?:api_key|access_token|refresh_token|password|secret|authorization)"\s*:\s*)"[^"]*"/gi,
    '$1"[redacted]"'
  );
}

/**
 * 写请求只发送一次；不在缺少幂等保证时自动重试。
 * @param request - 请求地址、方法、凭据和超时配置。
 * @returns 本次请求的原始 HTTP 响应。
 */
export async function requestOnce(request: HttpRequest): Promise<Response> {
  try {
    return await fetch(request.url, {
      method: request.method,
      headers: request.headers,
      body: request.body,
      signal: AbortSignal.timeout(request.timeoutMs),
    });
  } catch (error) {
    throw new APIError(
      request.path,
      `${request.method} failed after at most ${request.timeoutMs}ms: ${redactResponse(error instanceof Error ? error.message : String(error), request.headers)}`
    );
  }
}

/**
 * GET 最多两次；仅网络错误及 502/503/504 重试，保留最后一次错误。
 * @param request - 请求地址、方法、凭据和超时配置。
 * @returns 首次成功或最后一次尝试的 HTTP 响应。
 */
export async function requestRead(request: HttpRequest): Promise<Response> {
  try {
    const response = await requestOnce(request);
    if (![502, 503, 504].includes(response.status)) return response;
    await response.body?.cancel();
    process.stderr.write(
      `${JSON.stringify({ level: 'warn', event: 'http_retry', method: request.method, path: request.path, status: response.status, attempt: 2 })}\n`
    );
  } catch (error) {
    process.stderr.write(
      `${JSON.stringify({ level: 'warn', event: 'http_retry', method: request.method, path: request.path, attempt: 2, error: error instanceof Error ? error.message : String(error) })}\n`
    );
  }
  await delay(100);
  return requestOnce(request);
}

/**
 * 解析服务响应；空响应只允许明确的 204，畸形 JSON 必须报错。
 * @param response - 服务返回的 HTTP 响应。
 * @param request - 请求地址、方法、凭据和超时配置。
 * @returns 待协议层继续校验的 JSON 数据；204 返回空对象。
 */
export async function readResponse(response: Response, request: HttpRequest): Promise<unknown> {
  if (response.status === 204) return {};
  if (!response.ok) {
    const body = redactResponse(await response.text(), request.headers);
    const detail = `${request.method} ${request.path}: HTTP ${response.status}, response=${body}`;
    if (response.status === 401 || (response.status === 403 && 'X-API-Key' in request.headers))
      throw new AuthError(detail);
    if (response.status === 404) throw new NotFoundError(detail);
    throw new APIError(request.path, detail);
  }
  try {
    return await response.json();
  } catch {
    throw new APIError(
      request.path,
      `${request.method}: HTTP ${response.status}, response is not valid JSON`
    );
  }
}
