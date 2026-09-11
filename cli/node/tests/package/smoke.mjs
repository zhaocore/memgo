import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import { createServer } from 'node:http';
import os from 'node:os';
import path from 'node:path';

/** 仅依赖 Node 内置模块，可在无源码、无开发依赖的安装目录运行。 */
const entry = path.resolve(process.argv[2]);
const sender = path.resolve(path.dirname(entry), '../telemetry-sender.cjs');
const home = fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-package-'));
const requests = [];
const server = createServer(async (req, res) => {
  let body = '';
  for await (const chunk of req) body += chunk;
  requests.push({
    method: req.method,
    path: req.url,
    body: body ? JSON.parse(body) : undefined,
  });
  res.setHeader('Content-Type', 'application/json');
  if (req.url === '/v1/ping/') res.end('{"user_email":"test@example.test"}');
  else if (req.method === 'POST' && req.url === '/memories')
    res.end('{"results":[{"id":"one","memory":"hello","event":"ADD"}]}');
  else if (req.url === '/search' || (req.method === 'GET' && req.url?.startsWith('/memories?')))
    res.end('{"results":[{"id":"one","memory":"hello"}]}');
  else res.end('{"id":"one","memory":"hello"}');
});
await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
const baseUrl = `http://127.0.0.1:${server.address().port}`;
const env = Object.fromEntries(
  Object.entries(process.env).filter(
    ([key]) => !key.startsWith('MEMGO_') && !/CLAUDE|CURSOR|CODEX|GEMINI|COPILOT|WINDSURF/.test(key)
  )
);
/**
 * 参数数组保持原样，不经 shell 拼接。
 * @param {string} script - 需要执行的构建产物路径。
 * @param {string[]} args - 传递给 CLI 的参数。
 * @param {string | undefined} input - 传给标准输入的文本。
 * @returns {Promise<{stdout: string, stderr: string, code: number | null}>} 子进程的标准输出、标准错误和退出码。
 */
async function run(script, args, input) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [script, ...args], {
      cwd: home,
      env: {
        ...env,
        HOME: home,
        MEMGO_BASE_URL: baseUrl,
        MEMGO_TELEMETRY: 'false',
        NO_COLOR: '1',
      },
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
      reject(new Error('Package smoke timed out'));
    }, 15000);
    child.on('error', reject);
    child.on('close', (code) => {
      clearTimeout(timeout);
      resolve({ code, stdout, stderr });
    });
    child.stdin.end(input);
  });
}
try {
  assert.equal((await run(entry, ['--help'], undefined)).code, 0);
  const version = await run(entry, ['--version'], undefined);
  assert.match(version.stdout, /0\.2\.13/);
  assert.equal((await run(entry, ['version'], undefined)).stdout, version.stdout);
  assert.equal(JSON.parse((await run(entry, ['help', '--json'], undefined)).stdout).name, 'memgo');
  fs.writeFileSync(path.join(home, '.zshrc'), 'export MEMGO_API_KEY="old"\n');
  fs.mkdirSync(path.join(home, '.claude'));
  fs.writeFileSync(
    path.join(home, '.claude/settings.json'),
    JSON.stringify({ env: { MEMGO_API_KEY: 'old', OTHER: 'preserved' } })
  );
  const init = await run(
    entry,
    ['--json', 'init', '--api-key', 'package-key', '--user-id', 'alice', '--force'],
    undefined
  );
  assert.equal(init.code, 0, init.stderr || init.stdout);
  assert.equal(JSON.parse(init.stdout).command, 'init');
  assert.equal(
    fs.readFileSync(path.join(home, '.zshrc'), 'utf8'),
    'export MEMGO_API_KEY="package-key"\n'
  );
  assert.equal(
    JSON.parse(fs.readFileSync(path.join(home, '.claude/settings.json'), 'utf8')).env.OTHER,
    'preserved'
  );
  for (const args of [
    ['add', 'hello'],
    ['search', 'hello'],
    ['get', 'one'],
    ['list'],
    ['update', 'one', 'changed'],
    ['delete', 'one', '--force'],
  ]) {
    const result = await run(entry, ['--json', ...args], undefined);
    assert.equal(result.code, 0, result.stderr || result.stdout);
    assert.equal(JSON.parse(result.stdout).status, 'success');
  }
  const config = await run(entry, ['--json', 'config', 'show'], undefined);
  assert.ok(!config.stdout.includes('"package-key"'));
  assert.equal((await run(entry, ['search'], undefined)).code, 1);
  const telemetry = await run(
    sender,
    [],
    JSON.stringify({
      payload: {
        api_key: 'public-test',
        distinct_id: 'test@example.test',
        event: 'package_smoke',
        properties: {},
      },
      posthogHost: `${baseUrl}/capture`,
      needsEmail: false,
      memgoApiKey: '',
      memgoBaseUrl: baseUrl,
      configPath: path.join(home, '.memgo/config.json'),
      anonDistinctIdToAlias: 'anonymous-test',
    })
  );
  assert.equal(telemetry.code, 0, telemetry.stderr);
  assert.deepEqual(
    requests.filter((request) => request.path === '/capture').map((request) => request.body.event),
    ['$identify', 'package_smoke']
  );
  console.log(`${process.version}: installed CLI, CRUD, plugin sync and telemetry sender passed`);
} finally {
  server.closeAllConnections();
  await new Promise((resolve) => server.close(resolve));
  fs.rmSync(home, { recursive: true, force: true });
}
