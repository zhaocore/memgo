import { spawn } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { type CliResult, packageRoot } from '../helpers/cli.js';
import { type HttpFixture, type TestResponse, serve } from '../helpers/http-server.js';

let fixture: HttpFixture;
let home: string;
let failure: TestResponse | undefined;
/**
 * 让 HTTP 事件循环继续运行，异步调用真实构建产物。
 * @param args - 传递给 CLI 的参数。
 * @param input - 传给标准输入的文本。
 * @returns 编译后 CLI 的输出与退出码。
 */
function invoke(args: string[], input: string | undefined): Promise<CliResult> {
  return new Promise((resolve, reject) => {
    const env = Object.fromEntries(
      Object.entries(process.env).filter(
        ([key]) =>
          !key.startsWith('MEMGO_') && !/CLAUDE|CURSOR|CODEX|GEMINI|COPILOT|WINDSURF/.test(key)
      )
    );
    const executable = input === undefined ? process.execPath : '/bin/sh';
    const commandArgs =
      input === undefined
        ? [path.join(packageRoot, 'dist/index.js'), ...args]
        : [
          '-c',
          'cat | "$@"',
          'memgo-stdin',
          process.execPath,
          path.join(packageRoot, 'dist/index.js'),
          ...args,
        ];
    const child = spawn(executable, commandArgs, {
      env: {
        ...env,
        HOME: home,
        MEMGO_BASE_URL: fixture.url,
        MEMGO_TELEMETRY: 'false',
        NO_COLOR: '1',
      },
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (chunk) => {
      stdout += chunk;
    });
    child.stderr.on('data', (chunk) => {
      stderr += chunk;
    });
    const timeout = setTimeout(() => {
      child.kill();
      reject(new Error('CLI did not exit within 10 seconds'));
    }, 10000);
    child.on('error', reject);
    child.on('close', (code) => {
      clearTimeout(timeout);
      resolve({ stdout, stderr, exitCode: code ?? 1 });
    });
    child.stdin.end(input);
  });
}
beforeEach(async () => {
  home = fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-built-'));
  failure = undefined;
  fixture = await serve((request) => {
    if (failure) return failure;
    const memory = { id: 'memory-1', memory: 'hello' };
    if (request.path === '/api/v1/auth/agent_mode/')
      return {
        status: 200,
        body: JSON.stringify({
          api_key: 'local-key',
          default_user_id: 'agent-user',
          org_id: 'org',
          project_id: 'project',
        }),
      };
    if (request.path === '/api/v1/auth/email_code/verify/')
      return {
        status: 200,
        body: JSON.stringify({
          api_key: 'local-key',
          claimed: true,
          claimed_at: '2026-09-11T00:00:00Z',
        }),
      };
    if (request.path === '/v1/ping/')
      return {
        status: 200,
        body: JSON.stringify({ user_email: 'local@example.test' }),
      };
    if (request.path === '/search' || (request.path === '/memories' && request.method === 'GET'))
      return { status: 200, body: JSON.stringify({ results: [memory] }) };
    if (request.path === '/memories' && request.method === 'POST')
      return {
        status: 200,
        body: JSON.stringify({ results: [{ ...memory, event: 'ADD' }] }),
      };
    return { status: 200, body: JSON.stringify(memory) };
  });
});
afterEach(async () => {
  await fixture.close();
  fs.rmSync(home, { recursive: true, force: true });
});

async function initialize(): Promise<void> {
  const result = await invoke(
    ['--json', 'init', '--api-key', 'local-key', '--user-id', 'alice', '--force'],
    undefined
  );
  expect(result.exitCode, result.stderr || result.stdout).toBe(0);
  expect(JSON.parse(result.stdout).command).toBe('init');
}

describe('构建产物 CLI 验收', () => {
  it('配置保存、读取和 OSS CRUD 连通，JSON 不混入进度或 Spinner', async () => {
    await initialize();
    for (const args of [
      ['add', 'hello'],
      ['search', 'hello'],
      ['get', 'memory-1'],
      ['list'],
      ['update', 'memory-1', 'updated'],
      ['delete', 'memory-1', '--force'],
    ]) {
      const result = await invoke(['--json', ...args], undefined);
      expect(result.exitCode, result.stderr || result.stdout).toBe(0);
      expect(JSON.parse(result.stdout).status).toBe('success');
    }
    const config = await invoke(['--json', 'config', 'show'], undefined);
    expect(config.stdout).not.toContain('"local-key"');
    expect(fs.statSync(path.join(home, '.memgo/config.json')).mode & 0o777).toBe(0o600);
  });
  it('文件、stdin、空输入及带空格参数经真实进程验证', async () => {
    await initialize();
    const file = path.join(home, 'messages with spaces.json');
    fs.writeFileSync(file, JSON.stringify([{ role: 'user', content: 'file input' }]));
    expect((await invoke(['add', '--file', file, '-o', 'json'], undefined)).exitCode).toBe(0);
    expect((await invoke(['add', '-o', 'json'], 'piped input')).exitCode).toBe(0);
    expect((await invoke(['search', '-o', 'json'], 'piped query')).exitCode).toBe(0);
    expect((await invoke(['search'], '')).exitCode).toBe(1);
    expect(fixture.requests.some((request) => request.json?.query === 'piped query')).toBe(true);
  });
  it('批量导入 JSON 可解析，错误退出无原始调用栈', async () => {
    await initialize();
    const file = path.join(home, 'import.json');
    fs.writeFileSync(file, JSON.stringify([{ memory: 'one' }, { memory: 'two' }]));
    const result = await invoke(['--json', 'import', file], undefined);
    expect(result.exitCode).toBe(0);
    expect(JSON.parse(result.stdout).data).toEqual({ added: 2, failed: 0 });
    failure = { status: 401, body: '{"detail":"invalid key"}' };
    const failed = await invoke(['--json', 'get', 'memory-1'], undefined);
    expect(failed.exitCode).toBe(1);
    expect(JSON.parse(failed.stdout).status).toBe('error');
    expect(failed.stderr).not.toContain(' at ');
  });
  it('Agent Mode 初始化和邮箱认领保持原账号，机器输出不含密钥', async () => {
    const bootstrap = await invoke(['init', '--agent', '--json'], undefined);
    expect(bootstrap.exitCode, bootstrap.stderr).toBe(0);
    expect(JSON.parse(bootstrap.stdout).data.agent_mode).toBe(true);
    expect(bootstrap.stdout).not.toContain('local-key');
    const claim = await invoke(
      ['--json', 'init', '--email', 'local@example.test', '--code', '123456'],
      undefined
    );
    expect(claim.exitCode, claim.stderr).toBe(0);
    expect(JSON.parse(claim.stdout).data.agent_mode).toBe(false);
    expect(
      fixture.requests.find((request) => request.path.endsWith('verify/'))?.json?.agent_mode_api_key
    ).toBe('local-key');
  });
  it('项目级 dry-run 不执行删除；畸形消息输入明确失败', async () => {
    await initialize();
    const result = await invoke(
      ['--json', 'delete', '--all', '--project', '--dry-run', '--force'],
      undefined
    );
    expect(result.exitCode).toBe(1);
    expect(fixture.requests.some((request) => request.method === 'DELETE')).toBe(false);
    expect(
      (await invoke(['--json', 'add', '--messages', '{"invalid":true}'], undefined)).exitCode
    ).toBe(1);
  });
});

it.each(['email', 'agent'])('%s 初始化拒绝畸形凭据，不写入配置', async (mode) => {
  failure = { status: 200, body: '{"api_key":123}' };
  const args =
    mode === 'email'
      ? ['--json', 'init', '--email', 'local@example.test', '--code', '123456']
      : ['init', '--agent', '--json'];
  const result = await invoke(args, undefined);
  expect(result.exitCode).toBe(1);
  expect(JSON.parse(result.stdout).status).toBe('error');
  expect(fs.existsSync(path.join(home, '.memgo/config.json'))).toBe(false);
});

it('未知配置字段返回错误信封及非零退出码', async () => {
  for (const args of [
    ['config', 'get', 'missing'],
    ['config', 'set', 'missing', 'value'],
  ]) {
    const result = await invoke(['--json', ...args], undefined);
    expect(result.exitCode).toBe(1);
    expect(JSON.parse(result.stdout).status).toBe('error');
  }
});
