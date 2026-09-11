/** 校验密钥复用与插件同步边界；网络失败不能触发新账号，更新已有条目时保留其他内容。 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { pingKey } from '../../src/application/onboarding/key.js';
import { updateClaudeSettings, updateShellRc } from '../../src/integrations/plugin-sync.js';

// 密钥有效性判断。

describe('pingKey — network vs auth distinction', () => {
  const origFetch = globalThis.fetch;
  afterEach(() => {
    globalThis.fetch = origFetch;
    vi.restoreAllMocks();
  });

  it('returns true for 200', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 200 } as Response);
    await expect(pingKey('k', 'http://x', 5000)).resolves.toBe(true);
  });

  it('returns false for 401 (definitively invalid)', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 401 } as Response);
    await expect(pingKey('k', 'http://x', 5000)).resolves.toBe(false);
  });

  it('returns false for 403 (definitively invalid)', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 403 } as Response);
    await expect(pingKey('k', 'http://x', 5000)).resolves.toBe(false);
  });

  it('returns true for 5xx (transient upstream — prefer reuse)', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ status: 503 } as Response);
    await expect(pingKey('k', 'http://x', 5000)).resolves.toBe(true);
  });

  it('returns true on network error (prefer reuse over re-mint)', async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error('ECONNREFUSED'));
    await expect(pingKey('k', 'http://x', 5000)).resolves.toBe(true);
  });

  it('returns true on timeout (prefer reuse)', async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error('aborted'));
    await expect(pingKey('k', 'http://x', 5000)).resolves.toBe(true);
  });
});

// shell 配置同步。

describe('updateShellRc — exists-only contract', () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-test-'));
  });
  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it('updates existing export and preserves trailing newline', () => {
    const rc = path.join(tmpDir, '.zshrc');
    fs.writeFileSync(rc, 'export MEMGO_API_KEY="old"\n');
    expect(updateShellRc(rc, 'newkey')).toBe(true);
    expect(fs.readFileSync(rc, 'utf-8')).toBe('export MEMGO_API_KEY="newkey"\n');
  });

  it('does NOT create a new export when none exists', () => {
    const rc = path.join(tmpDir, '.zshrc');
    fs.writeFileSync(rc, "alias ll='ls -la'\n");
    expect(updateShellRc(rc, 'newkey')).toBe(false);
    expect(fs.readFileSync(rc, 'utf-8')).toBe("alias ll='ls -la'\n");
  });

  it('preserves surrounding content', () => {
    const rc = path.join(tmpDir, '.zshrc');
    const original =
      '# my zshrc\n' +
      "alias ll='ls -la'\n" +
      "export MEMGO_API_KEY='old'\n" +
      'export OTHER=keepme\n';
    fs.writeFileSync(rc, original);
    updateShellRc(rc, 'newkey');
    const after = fs.readFileSync(rc, 'utf-8');
    expect(after).toContain("alias ll='ls -la'\n");
    expect(after).toContain('export OTHER=keepme\n');
    expect(after).toContain('# my zshrc\n');
    expect(after).toContain('export MEMGO_API_KEY="newkey"\n');
  });

  it('is idempotent when value already matches', () => {
    const rc = path.join(tmpDir, '.zshrc');
    fs.writeFileSync(rc, 'export MEMGO_API_KEY="same"\n');
    expect(updateShellRc(rc, 'same')).toBe(false);
  });

  it('is a no-op for missing files', () => {
    const rc = path.join(tmpDir, '.zshrc'); // 文件尚不存在。
    expect(updateShellRc(rc, 'x')).toBe(false);
  });
});

// Claude 配置同步。

describe('updateClaudeSettings — never creates entries', () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-test-'));
  });
  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it('does not create env block when none exists', () => {
    const settings = path.join(tmpDir, 'settings.json');
    fs.writeFileSync(settings, JSON.stringify({ otherKey: 1 }));
    expect(updateClaudeSettings(settings, 'newkey')).toBe(false);
    expect(JSON.parse(fs.readFileSync(settings, 'utf-8'))).toEqual({
      otherKey: 1,
    });
  });

  it('does not create MEMGO_API_KEY entry in existing env block', () => {
    const settings = path.join(tmpDir, 'settings.json');
    fs.writeFileSync(settings, JSON.stringify({ env: { OTHER_KEY: 'x' } }));
    expect(updateClaudeSettings(settings, 'newkey')).toBe(false);
  });

  it('updates existing entry and preserves siblings', () => {
    const settings = path.join(tmpDir, 'settings.json');
    fs.writeFileSync(
      settings,
      JSON.stringify({ env: { MEMGO_API_KEY: 'old', OTHER: 'y' } }, null, 2)
    );
    expect(updateClaudeSettings(settings, 'fresh')).toBe(true);
    const data = JSON.parse(fs.readFileSync(settings, 'utf-8'));
    expect(data.env.MEMGO_API_KEY).toBe('fresh');
    expect(data.env.OTHER).toBe('y');
  });

  it('is idempotent when value already matches', () => {
    const settings = path.join(tmpDir, 'settings.json');
    fs.writeFileSync(settings, JSON.stringify({ env: { MEMGO_API_KEY: 'same' } }));
    expect(updateClaudeSettings(settings, 'same')).toBe(false);
  });

  it('is a no-op for malformed JSON', () => {
    const settings = path.join(tmpDir, 'settings.json');
    fs.writeFileSync(settings, '{ this is not json');
    expect(updateClaudeSettings(settings, 'x')).toBe(false);
  });
});
