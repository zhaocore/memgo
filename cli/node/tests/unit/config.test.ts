/** 配置管理测试。 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import {
  createDefaultConfig,
  getNestedValue,
  redactKey,
  setNestedValue,
} from '../../src/config/store.js';

// 测试使用临时配置目录。
let tmpDir: string;

beforeEach(() => {
  tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-test-'));

  // 清除配置环境变量。
  for (const key of Object.keys(process.env)) {
    if (key.startsWith('MEMGO_')) {
      delete process.env[key];
    }
  }
});

afterEach(() => {
  fs.rmSync(tmpDir, { recursive: true, force: true });
});

describe('redactKey', () => {
  it("returns '(not set)' for empty key", () => {
    expect(redactKey('')).toBe('(not set)');
  });

  it('redacts short key', () => {
    expect(redactKey('abc')).toBe('ab***');
  });

  it('redacts normal key', () => {
    const result = redactKey('m0-abcdefgh12345678');
    expect(result).toBe('m0-a...5678');
    expect(result).not.toContain('abcdefgh');
  });

  it('redacts exactly 8-char key as short', () => {
    expect(redactKey('12345678')).toBe('12***');
  });
});

describe('createDefaultConfig', () => {
  it('has correct defaults', () => {
    const config = createDefaultConfig();
    expect(config.platform.baseUrl).toBe('https://api.memgo.ai');
    expect(config.platform.apiKey).toBe('');
    expect(config.defaults.userId).toBe('');
  });
});

describe('getNestedValue', () => {
  it('gets platform.api_key', () => {
    const config = createDefaultConfig();
    config.platform.apiKey = 'test-key';
    expect(getNestedValue(config, 'platform.api_key')).toBe('test-key');
  });

  it('returns undefined for nonexistent key', () => {
    const config = createDefaultConfig();
    expect(getNestedValue(config, 'nonexistent.key')).toBeUndefined();
  });

  it('gets defaults.user_id', () => {
    const config = createDefaultConfig();
    config.defaults.userId = 'alice';
    expect(getNestedValue(config, 'defaults.user_id')).toBe('alice');
  });
});

describe('setNestedValue', () => {
  it('sets platform.api_key', () => {
    const config = createDefaultConfig();
    const updated = setNestedValue(config, 'platform.api_key', 'new-key');
    expect(config.platform.apiKey).toBe('');
    expect(updated.platform.apiKey).toBe('new-key');
  });

  it('returns false for nonexistent key', () => {
    const config = createDefaultConfig();
    expect(() => setNestedValue(config, 'nonexistent.key', 'val')).toThrow('Unknown config key');
  });

  it('sets defaults.user_id', () => {
    const config = createDefaultConfig();
    const updated = setNestedValue(config, 'defaults.user_id', 'bob');
    expect(updated.defaults.userId).toBe('bob');
  });
});
