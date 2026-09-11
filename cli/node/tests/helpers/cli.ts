import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
export interface CliResult {
  stdout: string;
  stderr: string;
  exitCode: number;
}
/**
 * 参数数组直接传给 Node，不经 shell 拼接；测试永不接触用户配置或真实遥测。
 * @param args - 传递给 CLI 的参数。
 * @param opts - 本次操作选项。
 * @param opts.home - 用于隔离配置的临时用户目录。
 * @param opts.env - 调用环境中的变量。
 * @param opts.input - 调用方提供的输入。
 * @returns 子进程的标准输出、标准错误和退出码。
 */
export function run(
  args: string[],
  opts: { home?: string; env?: Record<string, string>; input?: string }
): CliResult {
  const temporary = opts.home === undefined;
  const home = opts.home ?? fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-cli-'));
  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      ([key]) =>
        !key.startsWith('MEMGO_') && !/CLAUDE|CURSOR|CODEX|GEMINI|COPILOT|WINDSURF/.test(key)
    )
  );
  try {
    const result = spawnSync(process.execPath, [path.join(packageRoot, 'dist/index.js'), ...args], {
      cwd: packageRoot,
      env: {
        ...env,
        HOME: home,
        MEMGO_TELEMETRY: 'false',
        MEMGO_BASE_URL: 'http://127.0.0.1:1',
        NO_COLOR: '1',
        ...opts.env,
      },
      encoding: 'utf8',
      input: opts.input,
      timeout: 15000,
    });
    if (result.error) throw result.error;
    return {
      stdout: result.stdout,
      stderr: result.stderr,
      exitCode: result.status ?? 1,
    };
  } finally {
    if (temporary) fs.rmSync(home, { recursive: true, force: true });
  }
}
