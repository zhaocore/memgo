/** 独立遥测发送器；凭据只从 stdin 读取，不经进程参数。 */
'use strict';
const fs = require('node:fs');

/**
 * 遥测事件负载。
 * @typedef {{api_key: string, distinct_id: string, event: string, properties: Record<string, unknown>}} Payload
 */
/**
 * 子进程上下文。
 * @typedef {{payload: Payload, posthogHost: string, needsEmail: boolean, memgoApiKey: string, memgoBaseUrl: string, configPath: string, anonDistinctIdToAlias: string | null}} Context
 */

/**
 * 从标准输入读取并校验发送上下文。
 * @returns {Promise<Context>} 校验后的发送上下文。
 */
async function loadContext() {
  let text = '';
  process.stdin.setEncoding('utf8');
  for await (const chunk of process.stdin) text += chunk;
  const value = JSON.parse(text);
  if (
    !value ||
    typeof value !== 'object' ||
    !value.payload ||
    typeof value.payload.event !== 'string' ||
    typeof value.payload.api_key !== 'string' ||
    typeof value.payload.distinct_id !== 'string' ||
    typeof value.posthogHost !== 'string' ||
    typeof value.memgoBaseUrl !== 'string' ||
    typeof value.memgoApiKey !== 'string' ||
    typeof value.configPath !== 'string'
  )
    throw new Error('Invalid telemetry context');
  return value;
}
/**
 * 单次请求失败时保留 HTTP 状态，不输出凭据。
 * @param {string} url - 请求地址。
 * @param {string} method - HTTP 方法。
 * @param {Record<string, string>} headers - 请求头。
 * @param {string | undefined} body - 请求正文。
 * @returns {Promise<Response>} 成功的 HTTP 响应。
 */
async function request(url, method, headers, body) {
  const response = await fetch(url, {
    method,
    headers,
    body,
    signal: AbortSignal.timeout(10000),
  });
  if (!response.ok) throw new Error(`Telemetry ${method} failed: HTTP ${response.status}`);
  return response;
}
/**
 * 获取账号邮箱并缓存，构造更新身份后的事件负载。
 * @param {Context} context - 发送上下文。
 * @returns {Promise<Payload>} 已解析身份的负载；无需查询时返回原负载。
 */
async function resolveEmail(context) {
  if (!context.needsEmail || !context.memgoApiKey) return context.payload;
  const response = await request(
    `${context.memgoBaseUrl.replace(/\/+$/, '')}/v1/ping/`,
    'GET',
    { Authorization: `Token ${context.memgoApiKey}` },
    undefined
  );
  const data = await response.json();
  if (typeof data.user_email !== 'string')
    throw new Error('Telemetry ping response missing user_email');
  const config = JSON.parse(fs.readFileSync(context.configPath, 'utf8'));
  fs.writeFileSync(
    context.configPath,
    JSON.stringify(
      {
        ...config,
        platform: { ...config.platform, user_email: data.user_email },
      },
      null,
      2
    )
  );
  return { ...context.payload, distinct_id: data.user_email };
}
/**
 * 发送事件，不重试可能造成重复计数的写请求。
 * @param {string} host - 遥测接收地址。
 * @param {Payload} payload - 待发送的事件。
 * @returns {Promise<void>} 事件发送完成。
 */
async function send(host, payload) {
  await request(host, 'POST', { 'Content-Type': 'application/json' }, JSON.stringify(payload));
}
/**
 * 先关联匿名身份，再发送原事件。
 * @returns {Promise<void>} 本次发送流程完成。
 */
async function main() {
  const context = await loadContext();
  let payload = context.payload;
  try {
    payload = await resolveEmail(context);
  } catch {
    process.stderr.write('Telemetry identity lookup failed; retaining the existing identity.\n');
  }
  if (context.anonDistinctIdToAlias)
    await send(context.posthogHost, {
      api_key: payload.api_key,
      distinct_id: payload.distinct_id,
      event: '$identify',
      properties: {
        $anon_distinct_id: context.anonDistinctIdToAlias,
        $lib: 'posthog-node',
      },
    });
  await send(context.posthogHost, payload);
}
main().catch(() => {
  process.stderr.write('Telemetry delivery failed.\n');
  process.exitCode = 1;
});
