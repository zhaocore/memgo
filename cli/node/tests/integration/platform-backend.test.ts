import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { PlatformBackend } from '../../src/backend/platform.js';
import { createDefaultConfig } from '../../src/config/defaults.js';
import { captureNotice, isAgentMode } from '../../src/runtime/state.js';
import { takeNotice } from '../../src/runtime/state.js';
import { type HttpFixture, type TestResponse, serve } from '../helpers/http-server.js';

let fixture: HttpFixture;
let backend: PlatformBackend;
let response: TestResponse;
beforeEach(async () => {
  response = { status: 200, body: JSON.stringify({ results: [] }) };
  fixture = await serve(() => response);
  backend = new PlatformBackend(
    {
      ...createDefaultConfig().platform,
      baseUrl: fixture.url,
      apiKey: 'test-key',
    },
    {
      callerType: () => (isAgentMode() ? 'agent' : 'user'),
      notice: captureNotice,
    }
  );
});
afterEach(async () => fixture.close());

describe('Platform HTTP 合同', () => {
  it('传递添加字段及 Token 鉴权，不丢弃已有参数', async () => {
    await backend.add('hello', undefined, {
      userId: 'alice',
      metadata: { source: 'test' },
      expires: '2099-01-01',
      customInstructions: 'Extract preferences',
      agentCustomInstructions: 'Extract tools',
      customCategories: [{ prefs: 'preferences' }],
      structuredDataSchema: { type: 'object' },
      timestamp: 1700000000,
    });
    expect(fixture.requests[0]).toMatchObject({
      method: 'POST',
      path: '/v3/memories/add/',
      headers: {
        authorization: 'Token test-key',
        'x-memgo-caller-type': 'user',
      },
      json: {
        messages: [{ role: 'user', content: 'hello' }],
        user_id: 'alice',
        metadata: { source: 'test' },
        expiration_date: '2099-01-01',
        custom_instructions: 'Extract preferences',
        agent_custom_instructions: 'Extract tools',
        custom_categories: [{ prefs: 'preferences' }],
        structured_data_schema: { type: 'object' },
        timestamp: 1700000000,
        source: 'CLI',
      },
    });
  });
  it('缺失的可选字段不进入添加请求', async () => {
    await backend.add('hello', undefined, { userId: 'alice' });
    expect(fixture.requests[0].json).toEqual({
      messages: [{ role: 'user', content: 'hello' }],
      user_id: 'alice',
      source: 'CLI',
    });
  });
  it('搜索字段及预构造过滤器保持语义', async () => {
    await backend.search('query', {
      showExpired: true,
      referenceDate: '2024-01-01',
      latestOnly: true,
      keyword: true,
      fields: ['memory', 'score'],
      filters: { OR: [{ user_id: 'alice' }] },
    });
    expect(fixture.requests[0].json).toMatchObject({
      show_expired: true,
      reference_date: '2024-01-01',
      latest_only: true,
      keyword_search: true,
      fields: ['memory', 'score'],
      filters: { OR: [{ user_id: 'alice' }] },
    });
    await backend.search('query', {});
    expect(fixture.requests[1].json).not.toHaveProperty('keyword_search');
    expect(fixture.requests[1].json).not.toHaveProperty('fields');
  });
  it('列表顶层字段及分页查询参数保持独立', async () => {
    await backend.listMemories({
      userId: 'alice',
      showExpired: true,
      latestOnly: true,
      page: 2,
      pageSize: 5,
    });
    expect(fixture.requests[0]).toMatchObject({
      params: { page: '2', page_size: '5' },
      json: { show_expired: true, latest_only: true },
    });
    expect(fixture.requests[0].json?.filters).not.toHaveProperty('show_expired');
  });
  it('更新和删除的协议字段位置不变', async () => {
    await backend.update('mem-123', undefined, undefined, {
      expirationDate: '2099-01-01',
      timestamp: 1700000000,
    });
    expect(fixture.requests[0].json).toMatchObject({
      expiration_date: '2099-01-01',
      timestamp: 1700000000,
    });
    await backend.delete('mem-123', { deleteLinked: true });
    expect(fixture.requests[1].params.delete_linked).toBe('true');
    expect(fixture.requests[1].json).toBeUndefined();
  });
  it('多实体删除保留所有结果，空范围拒绝执行', async () => {
    response.body = JSON.stringify({ message: 'deleted' });
    expect(await backend.deleteEntities({ userId: 'alice', agentId: 'bob' })).toEqual({
      user: { message: 'deleted' },
      agent: { message: 'deleted' },
    });
    expect(fixture.requests.map((r) => r.path)).toEqual([
      '/v2/entities/user/alice/',
      '/v2/entities/agent/bob/',
    ]);
    await expect(backend.deleteEntities({})).rejects.toThrow('At least one entity ID');
  });
  it('对记忆、实体、事件 ID 编码，避免路径注入', async () => {
    await backend.get('mem/a?b#c');
    await backend.update('mem/a?b#c', 'updated', undefined, {});
    await backend.delete('mem/a?b#c', {});
    await backend.deleteEntities({ userId: 'org/team?active#frag' });
    await backend.getEvent('evt/a?b#c');
    expect(fixture.requests.map((r) => r.path)).toEqual([
      '/v1/memories/mem%2Fa%3Fb%23c/',
      '/v1/memories/mem%2Fa%3Fb%23c/',
      '/v1/memories/mem%2Fa%3Fb%23c/',
      '/v2/entities/user/org%2Fteam%3Factive%23frag/',
      '/v1/event/evt%2Fa%3Fb%23c/',
    ]);
  });
  it('提取提示但保留响应的其余字段', async () => {
    response.body = JSON.stringify({
      id: 'memory',
      memgo_notice: 'claim account',
    });
    expect(await backend.get('memory')).toEqual({ id: 'memory' });
    expect(takeNotice()).toBe('claim account');
    expect(takeNotice()).toBe('');
  });
  it('畸形 JSON 和失败响应明确报错，凭据不回显', async () => {
    response.body = 'invalid JSON';
    await expect(backend.get('memory')).rejects.toThrow('not valid JSON');
    response = {
      status: 401,
      body: JSON.stringify({ api_key: 'test-key', detail: 'Token test-key' }),
    };
    const error = await backend.get('memory').catch((e: Error) => e);
    expect(error).toBeInstanceOf(Error);
    expect(String(error)).toContain('HTTP 401');
    expect(String(error)).not.toContain('test-key');
  });
  it('GET 瞬时错误重试一次，写请求不重试', async () => {
    response = { status: 503, body: '{"detail":"unavailable"}' };
    await expect(backend.get('memory')).rejects.toThrow('HTTP 503');
    expect(fixture.requests).toHaveLength(2);
    await expect(backend.add('hello', undefined, {})).rejects.toThrow('HTTP 503');
    expect(fixture.requests).toHaveLength(3);
  });
});
