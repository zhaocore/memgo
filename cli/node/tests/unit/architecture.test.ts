import { describe, expect, it } from 'vitest';
import { resolveIds } from '../../src/application/entity-ids.js';
import { createProgram } from '../../src/cli/program.js';
import { createDefaultConfig } from '../../src/config/defaults.js';
import { resolveConfig } from '../../src/config/parse.js';
import {
  createInvocationState,
  getCurrentCommand,
  setCurrentCommand,
  withInvocation,
} from '../../src/runtime/state.js';

describe('模块边界和配置规则', () => {
  it('创建命令树没有解析参数或执行命令的副作用，树可独立扩展', () => {
    const first = createProgram();
    const second = createProgram();
    first.command('test-extension');
    expect(second.commands.some((command) => command.name() === 'test-extension')).toBe(false);
    expect(second.commands.map((command) => command.name())).toEqual([
      'init',
      'identify',
      'whoami',
      'agent-rush',
      'add',
      'search',
      'get',
      'list',
      'update',
      'delete',
      'config',
      'entity',
      'event',
      'status',
      'version',
      'import',
      'help',
    ]);
  });
  it('显式实体范围不混入其他默认实体', () => {
    const config = createDefaultConfig();
    config.defaults.agentId = 'default-agent';
    expect(resolveIds(config, { userId: 'alice' })).toEqual({
      userId: 'alice',
      agentId: undefined,
      appId: undefined,
      runId: undefined,
    });
    expect(config.defaults.agentId).toBe('default-agent');
  });
  it('环境覆盖文件，未知字段忽略，已知字段类型错误明确失败', () => {
    const raw = {
      platform: { api_key: 'file-key', extra: 42 },
      defaults: { user_id: 'file-user' },
    };
    expect(resolveConfig(raw, { MEMGO_API_KEY: 'env-key' }).platform.apiKey).toBe('env-key');
    expect(raw.platform.api_key).toBe('file-key');
    expect(() => resolveConfig({ platform: { api_key: 123 } }, {})).toThrow('platform.api_key');
  });
  it('并发调用的命令状态独立', async () => {
    const result = await Promise.all(
      ['first', 'second'].map((name) =>
        withInvocation(createInvocationState(), async () => {
          setCurrentCommand(name);
          await new Promise((resolve) => setTimeout(resolve, 2));
          return getCurrentCommand();
        })
      )
    );
    expect(result).toEqual(['first', 'second']);
  });
});
