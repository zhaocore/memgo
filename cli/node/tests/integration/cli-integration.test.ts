import { run } from '../helpers/cli.js';
/** 通过子进程验证完整 CLI 行为。 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

describe('CLI Integration — help and version', () => {
  it('shows help with --help', () => {
    const result = run(['--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('memgo');
    expect(result.stdout).toContain('add');
    expect(result.stdout).toContain('search');
  });

  it('prints the version with --version', () => {
    const flag = run(['--version'], {});
    expect(flag.exitCode).toBe(0);
    expect(flag.stdout).toContain('MemGo');
  });

  it('version subcommand output matches --version output byte-for-byte', () => {
    const flag = run(['--version'], {});
    const cmd = run(['version'], {});
    expect(cmd.exitCode).toBe(0);
    expect(cmd.stdout).toBe(flag.stdout);
  });

  it.each([
    ['help', '--json'],
    ['--json', 'help'],
    ['--agent', 'help'],
  ])('%s %s produces valid JSON', (...args) => {
    const result = run(args, {});
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.stdout);
    // 兼容既有规范中的两种名称位置。
    const name = parsed.name ?? parsed.cli?.name;
    expect(name).toBe('memgo');
  });

  it('shows add help', () => {
    const result = run(['add', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('user-id');
    expect(result.stdout).toContain('messages');
  });

  it('shows search help', () => {
    const result = run(['search', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('top-k');
  });

  it('shows list help', () => {
    const result = run(['list', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('page-size');
  });

  it('shows delete help with --all, --entity, --project', () => {
    const result = run(['delete', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--all');
    expect(result.stdout).toContain('--entity');
    expect(result.stdout).toContain('--project');
    expect(result.stdout).toContain('--force');
    expect(result.stdout.toLowerCase()).toContain('memory');
  });

  it('delete with no args errors', () => {
    const result = run(['delete'], {});
    expect(result.exitCode).not.toBe(0);
    const combined = result.stdout + result.stderr;
    expect(combined).toContain('--all');
  });

  it('shows entity list help', () => {
    const result = run(['entity', 'list', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout.toLowerCase()).toContain('entitytype');
  });

  it('shows entity delete help', () => {
    const result = run(['entity', 'delete', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--user-id');
    expect(result.stdout).toContain('--force');
  });

  it('shows import help', () => {
    const result = run(['import', '--help'], {});
    expect(result.exitCode).toBe(0);
  });

  it('add help has --output flag', () => {
    const result = run(['add', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--output');
  });

  it('search help has --rerank flag', () => {
    const result = run(['search', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--rerank');
  });

  it('search help documents the --filter JSON shape with an example', () => {
    const result = run(['search', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('AND');
    expect(result.stdout).toContain('categories');
  });

  it('list help has --category flag', () => {
    const result = run(['list', '--help'], {});
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('--category');
  });
});

describe('CLI Integration — isolated (clean home)', () => {
  function cleanHome(): string {
    return fs.mkdtempSync(path.join(os.tmpdir(), 'memgo-test-'));
  }

  it('add without API key errors', () => {
    const home = cleanHome();
    const result = run(['add', 'test', '--user-id', 'alice'], { home });
    expect(result.exitCode).not.toBe(0);
    const combined = result.stdout + result.stderr;
    expect(combined.toLowerCase()).toMatch(/api.key|error/i);
    fs.rmSync(home, { recursive: true, force: true });
  });

  it('config show works with clean home', () => {
    const home = cleanHome();
    const result = run(['config', 'show'], { home });
    expect(result.exitCode).toBe(0);
    fs.rmSync(home, { recursive: true, force: true });
  });
});
