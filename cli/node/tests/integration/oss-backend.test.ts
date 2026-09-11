import { afterEach, beforeEach, expect, it } from 'vitest';
import { OSSBackend } from '../../src/backend/oss.js';
import { createDefaultConfig } from '../../src/config/defaults.js';
import { type HttpFixture, serve } from '../helpers/http-server.js';
let fixture: HttpFixture;
let backend: OSSBackend;
beforeEach(async () => {
  fixture = await serve((request) => ({
    status: 200,
    body: JSON.stringify(
      request.path === '/search' || (request.method === 'GET' && request.path === '/memories')
        ? { results: [] }
        : { id: 'memory' }
    ),
  }));
  backend = new OSSBackend({
    ...createDefaultConfig().platform,
    baseUrl: fixture.url,
    apiKey: 'local-key',
  });
});
afterEach(async () => fixture.close());
it('OSS 使用独立鉴权、路径和请求映射', async () => {
  await backend.add('hello', undefined, {
    userId: 'alice',
    infer: false,
    customInstructions: 'extract',
  });
  await backend.search('hello', { userId: 'alice' });
  await backend.listMemories({ userId: 'alice', pageSize: 5 });
  await backend.update('memory', undefined, { source: 'test' }, {});
  expect(fixture.requests[0]).toMatchObject({
    path: '/memories',
    headers: { 'x-api-key': 'local-key' },
    json: { user_id: 'alice', infer: false, prompt: 'extract' },
  });
  expect(fixture.requests[0].headers.authorization).toBeUndefined();
  expect(fixture.requests[1]).toMatchObject({
    path: '/search',
    json: { query: 'hello', filters: { user_id: 'alice' } },
  });
  expect(fixture.requests[2].params).toEqual({ user_id: 'alice', top_k: '5' });
  expect(fixture.requests[3].json).toEqual({ metadata: { source: 'test' } });
});
it('平台专属选项在发送请求前明确失败', async () => {
  await expect(backend.add('hello', undefined, { appId: 'app' })).rejects.toThrow('not supported');
  await expect(backend.search('hello', { keyword: true })).rejects.toThrow('not supported');
  await expect(backend.listEvents()).rejects.toThrow('not supported');
  expect(fixture.requests).toHaveLength(0);
});
