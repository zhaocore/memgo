import { sendEmailCode, verifyEmailCode } from '../../backend/auth.js';
import { type JsonObject, jsonObject } from '../../backend/json.js';
import { colors, printError, printSuccess } from '../../output/branding.js';
import { exit } from '../../runtime/exit.js';
import { promptLine } from '../../runtime/prompt.js';
const { brand } = colors;
const EMAIL_RE = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;
/**
 * 检查邮箱格式，非法输入以失败状态结束命令。
 * @param email - 用于登录或认领的邮箱。
 */
export function validateEmail(email: string): void {
  if (!EMAIL_RE.test(email)) {
    printError(`Invalid email address: ${JSON.stringify(email)}`);
    exit(1);
  }
}

/**
 * 发送或验证邮箱验证码并取得登录响应。
 * @param email - 用于登录或认领的邮箱。
 * @param code - 邮箱验证码。
 * @param baseUrl - 服务基础地址。
 * @returns 邮箱验证成功后的登录响应对象。
 */
export async function emailLogin(
  email: string,
  code: string | undefined,
  baseUrl: string
): Promise<JsonObject> {
  const url = baseUrl.replace(/\/+$/, '');
  let codeValue = code;

  if (!codeValue) {
    const resp = await sendEmailCode(url, email);
    if (resp.status === 429) {
      printError('Too many attempts. Try again in a few minutes.');
      exit(1);
    }
    if (!resp.ok) {
      let detail: string;
      try {
        const body = (await resp.json()) as JsonObject;
        detail = (body.error ?? body.detail ?? resp.statusText) as string;
      } catch {
        detail = resp.statusText;
      }
      printError(`Failed to send code: ${detail}`);
      exit(1);
    }

    printSuccess('Verification code sent! Check your email.');

    if (!process.stdin.isTTY) {
      printError(
        'No --code provided and terminal is non-interactive.',
        'Run: memgo init --email <email> --code <code>'
      );
      exit(1);
    }

    console.log();
    const entered = await promptLine(`  ${brand('Verification Code')}`);
    if (!entered) {
      printError('Code is required.');
      exit(1);
    }
    codeValue = entered;
  }

  const verifyResp = await verifyEmailCode(url, email, codeValue.trim(), undefined);
  if (verifyResp.status === 429) {
    printError('Too many attempts. Try again in a few minutes.');
    exit(1);
  }
  if (!verifyResp.ok) {
    let detail: string;
    try {
      const body = (await verifyResp.json()) as JsonObject;
      detail = (body.error ?? body.detail ?? verifyResp.statusText) as string;
    } catch {
      detail = verifyResp.statusText;
    }
    printError(`Verification failed: ${detail}`);
    exit(1);
  }

  const result = jsonObject(await verifyResp.json());
  if (typeof result.api_key !== 'string' || !result.api_key)
    throw new Error('Email verification response missing a valid api_key');
  return result;
}
