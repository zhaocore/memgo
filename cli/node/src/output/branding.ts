/** 终端品牌颜色和字符图案。 */

import chalk from 'chalk';
import ora from 'ora';
import { getCurrentCommand, isAgentMode } from '../runtime/state.js';
import { CLI_VERSION } from '../version.js';

/** MOMGO CLI 大号字符标志。 */
export const LOGO = `
█   █  ███  █   █  ███   ███     ████ █     █████
██ ██ █   █ ██ ██ █     █   █   █     █       █  
█ █ █ █   █ █ █ █ █ ███ █   █   █     █       █  
█   █ █   █ █   █ █   █ █   █   █     █       █  
█   █  ███  █   █  ███   ███     ████ █████ █████
`;

export const LOGO_MINI = '◆ memgo';
export const TAGLINE = 'The Memory Layer for AI Agents';

export const BRAND_COLOR = '#f472b6';
export const ACCENT_COLOR = '#67e8f9';
export const SUCCESS_COLOR = '#34d399';
export const ERROR_COLOR = '#fb7185';
export const WARNING_COLOR = '#facc15';
export const DIM_COLOR = '#94a3b8';

const brand = chalk.hex(BRAND_COLOR);
const accent = chalk.hex(ACCENT_COLOR);
const success = chalk.hex(SUCCESS_COLOR);
const error = chalk.hex(ERROR_COLOR);
const warning = chalk.hex(WARNING_COLOR);
const dim = chalk.hex(DIM_COLOR);

/**
 * 根据 TTY 和 NO_COLOR 选择终端符号，非交互环境使用普通字符。
 * @param fancy - 终端支持样式时使用的符号。
 * @param plain - 普通文本符号。
 * @returns 适合当前终端的符号。
 */
export function sym(fancy: string, plain: string): string {
  if (!process.stdout.isTTY || process.env.NO_COLOR) return plain;
  return fancy;
}

/**
 * 在人类交互模式下输出 CLI 品牌面板。
 */
export function printBanner(): void {
  if (isAgentMode()) return;
  const pad = 3; // 两侧水平留白。
  const logoLines = LOGO.trimEnd().split('\n');
  const tagline = `  ${TAGLINE}`;
  const subtitle = `Node.js SDK · v${CLI_VERSION}`;
  const contentLines = ['', ...logoLines, '', tagline, ''];

  // 按最长内容及两侧留白计算内部宽度。
  const maxContent = Math.max(...contentLines.map((l) => l.length));
  const innerWidth = maxContent + pad * 2;
  const totalWidth = innerWidth + 2; // 加上两侧边框。

  const topBorder = brand(`╭${'─'.repeat(totalWidth - 2)}╮`);
  const subtitleFill = totalWidth - 2 - subtitle.length - 3; // 标题前后分隔字符占三列。
  const bottomBorder = brand(`╰${'─'.repeat(subtitleFill)} ${dim(subtitle)} ${'─'}╯`);

  const body = contentLines.map((line) => {
    const rightPad = innerWidth - pad - line.length;
    return `${brand('│')}${' '.repeat(pad)}${brand.bold(line)}${' '.repeat(Math.max(rightPad, 0))}${brand('│')}`;
  });
  // 标语使用强调色。
  const taglineIdx = body.length - 2; // 标语位于尾部空行之前。
  const taglineRightPad = innerWidth - pad - tagline.length;
  body[taglineIdx] =
    `${brand('│')}${' '.repeat(pad)}${accent(tagline)}${' '.repeat(Math.max(taglineRightPad, 0))}${brand('│')}`;

  console.log(topBorder);
  for (const line of body) console.log(line);
  console.log(bottomBorder);
}

/**
 * 在人类交互模式下输出成功提示。
 * @param message - 需要展示的消息。
 */
export function printSuccess(message: string): void {
  if (isAgentMode()) return;
  console.log(`${success(sym('✓', '[ok]'))} ${message}`);
}

/**
 * 按本次输出模式显示错误及可选修复提示。
 * @param message - 需要展示的消息。
 * @param hint - 可选的故障处理提示。
 */
export function printError(message: string, hint?: string): void {
  if (isAgentMode()) {
    const envelope = {
      status: 'error',
      command: getCurrentCommand(),
      error: message,
      data: null,
    };
    console.log(JSON.stringify(envelope));
    return;
  }
  console.error(`${error(`${sym('✗', '[error]')} Error:`)} ${message}`);
  const resolvedHint =
    hint ??
    (message.includes('Authentication failed')
      ? `Run ${brand('memgo init')} to reconfigure your API key · https://app.memgo.ai/dashboard/api-keys?utm_source=oss&utm_medium=cli-node`
      : undefined);
  if (resolvedHint) {
    console.error(`  ${dim(resolvedHint)}`);
  }
}

/**
 * 向标准错误输出警告。
 * @param message - 需要展示的消息。
 */
export function printWarning(message: string): void {
  console.error(`${warning(sym('⚠', '[warn]'))} ${message}`);
}

/**
 * 在人类交互模式下向标准错误输出提示。
 * @param message - 需要展示的消息。
 */
export function printInfo(message: string): void {
  if (isAgentMode()) return;
  console.error(`${brand(sym('◆', '*'))} ${message}`);
}

/**
 * 展示已指定的实体范围。
 * @param ids - 当前实体范围。
 */
export function printScope(ids: Record<string, string | undefined>): void {
  if (isAgentMode()) return;
  const parts: string[] = [];
  for (const [key, val] of Object.entries(ids)) {
    if (val) {
      parts.push(`${key}=${val}`);
    }
  }
  if (parts.length > 0) {
    console.error(`  ${dim(`Scope: ${parts.join(', ')}`)}`);
  }
}

export interface TimedStatusContext {
  successMsg: string;
  errorMsg: string;
}

/**
 * 执行异步操作并计时；确保成功和失败路径都停止 Spinner。
 * @param message - 需要展示的消息。
 * @param fn - 需要计时并展示状态的异步任务。
 * @returns 异步任务的返回值。
 */
export async function timedStatus<T>(
  message: string,
  fn: (ctx: TimedStatusContext) => Promise<T>
): Promise<T> {
  if (isAgentMode()) {
    const ctx: TimedStatusContext = { successMsg: '', errorMsg: '' };
    return fn(ctx);
  }
  const ctx: TimedStatusContext = { successMsg: '', errorMsg: '' };
  const spinner = ora({
    text: dim(message),
    color: 'yellow',
    stream: process.stderr,
  }).start();
  const start = performance.now();

  try {
    const result = await fn(ctx);
    const elapsed = ((performance.now() - start) / 1000).toFixed(2);
    spinner.stop();
    if (ctx.successMsg) {
      console.error(`${success('✓')} ${ctx.successMsg} (${elapsed}s)`);
    }
    return result;
  } catch (err) {
    const elapsed = ((performance.now() - start) / 1000).toFixed(2);
    spinner.stop();
    if (ctx.errorMsg) {
      printError(`${ctx.errorMsg} (${elapsed}s)`);
    }
    throw err;
  }
}

/** 供格式化模块复用的品牌颜色。 */
export const colors = { brand, accent, success, error, warning, dim };
