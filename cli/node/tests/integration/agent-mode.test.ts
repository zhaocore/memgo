import { run } from '../helpers/cli.js';
/** 校验 Agent Mode 命令参数合同；网络流程另由集成测试覆盖，不修改 Python CLI。 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

function cleanHome(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-agent-'));
}

describe('init flag surface', () => {
  it('init --help lists --agent', () => {
    const result = run(['init', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--agent');
  });

  it('init --help describes Agent Mode', () => {
    const result = run(['init', '--help'], {});
    expect(result.exitCode).toBe(0);
    // 帮助须明确解释 --agent 的初始化作用。

    expect(
      result.stdout.includes('Agent Mode') || result.stdout.toLowerCase().includes('unattended')
    ).toBe(true);
  });

  it('init --help lists --source', () => {
    const result = run(['init', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--source');
  });

  it('init --help lists --email and --code', () => {
    const result = run(['init', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--email');
    expect(result.stdout).toContain('--code');
  });
});

describe('argv preprocessing — --agent reaches init subcommand', () => {
  // 验证根级 JSON 别名不会吞掉 init 的 --agent 参数。

  it('init --agent triggers bootstrap branch (not the wizard)', () => {
    const home = cleanHome();
    const result = run(['init', '--agent'], {
      home,
      env: {
        MEMGO_BASE_URL: 'http://127.0.0.1:1', // 不连接真实服务。
        FORCE_COLOR: '0',
      },
    });
    const combined = (result.stdout + result.stderr).toLowerCase();
    // 错误须来自初始化请求，不能落入交互向导。

    expect(
      combined.includes('agent') ||
        combined.includes('connect') ||
        combined.includes('network') ||
        combined.includes('fetch') ||
        combined.includes('bootstrap')
    ).toBe(true);
    fs.rmSync(home, { recursive: true, force: true });
  });
});

describe('JSON envelope on network failure', () => {
  it('init --agent --json does not leak a stack trace when backend is unreachable', () => {
    const home = cleanHome();
    const result = run(['init', '--agent', '--json'], {
      home,
      env: {
        MEMGO_BASE_URL: 'http://127.0.0.1:1',
        FORCE_COLOR: '0',
      },
    });
    const combined = result.stdout + result.stderr;
    // 不泄露原始 Node 调用栈。
    expect(combined).not.toMatch(/at \w+\s*\(.+\.ts:\d+/);
    expect(combined).not.toContain('UnhandledPromiseRejection');
    expect(result.exitCode).not.toBe(0);
    fs.rmSync(home, { recursive: true, force: true });
  });
});

describe('top-level help lists init', () => {
  // 顶层帮助须能发现 init 入口。

  it('--help lists init', () => {
    const result = run(['--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('init');
  });
});
