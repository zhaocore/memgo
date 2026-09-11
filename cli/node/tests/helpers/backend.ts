/** 测试共用后端接口桩。 */

import { vi } from 'vitest';
import type { Backend } from '../../src/backend/index.js';

/**
 * 创建覆盖所有后端方法的接口桩。
 * @returns 带可配置调用记录的测试后端。
 */
export function createMockBackend(): Backend {
  return {
    ping: vi.fn().mockResolvedValue({}),
    add: vi.fn().mockResolvedValue({
      results: [
        {
          id: 'abc-123-def-456',
          memory: 'User prefers dark mode',
          event: 'ADD',
        },
      ],
    }),

    search: vi.fn().mockResolvedValue([
      {
        id: 'abc-123-def-456',
        memory: 'User prefers dark mode',
        score: 0.92,
        created_at: '2026-02-15T10:30:00Z',
        categories: ['preferences'],
      },
      {
        id: 'ghi-789-jkl-012',
        memory: 'User uses vim keybindings',
        score: 0.78,
        created_at: '2026-03-01T14:00:00Z',
        categories: ['tools'],
      },
    ]),

    get: vi.fn().mockResolvedValue({
      id: 'abc-123-def-456',
      memory: 'User prefers dark mode',
      created_at: '2026-02-15T10:30:00Z',
      updated_at: '2026-02-20T08:00:00Z',
      metadata: { source: 'onboarding' },
      categories: ['preferences'],
    }),

    listMemories: vi.fn().mockResolvedValue([
      {
        id: 'abc-123-def-456',
        memory: 'User prefers dark mode',
        created_at: '2026-02-15T10:30:00Z',
        categories: ['preferences'],
      },
      {
        id: 'ghi-789-jkl-012',
        memory: 'User uses vim keybindings',
        created_at: '2026-03-01T14:00:00Z',
        categories: ['tools'],
      },
    ]),

    update: vi.fn().mockResolvedValue({ id: 'abc-123-def-456', memory: 'Updated memory' }),
    delete: vi.fn().mockResolvedValue({ status: 'deleted' }),
    status: vi.fn().mockResolvedValue({
      connected: true,
      backend: 'platform',
      base_url: 'https://api.memgo.ai',
    }),
    deleteEntities: vi.fn().mockResolvedValue({ message: 'Entity deleted' }),
    entities: vi.fn().mockResolvedValue([
      { name: 'alice', count: 5 },
      { name: 'bob', count: 3 },
    ]),
    listEvents: vi.fn().mockResolvedValue([
      {
        id: 'evt-abc-123-def-456',
        event_type: 'ADD',
        status: 'SUCCEEDED',
        graph_status: null,
        latency: 1234.5,
        created_at: '2026-04-01T10:00:00Z',
        updated_at: '2026-04-01T10:00:01Z',
      },
      {
        id: 'evt-def-456-ghi-789',
        event_type: 'SEARCH',
        status: 'PENDING',
        graph_status: null,
        latency: null,
        created_at: '2026-04-01T10:01:00Z',
        updated_at: '2026-04-01T10:01:00Z',
      },
    ]),
    getEvent: vi.fn().mockResolvedValue({
      id: 'evt-abc-123-def-456',
      event_type: 'ADD',
      status: 'SUCCEEDED',
      graph_status: 'SUCCEEDED',
      latency: 1234.5,
      created_at: '2026-04-01T10:00:00Z',
      updated_at: '2026-04-01T10:00:01Z',
      results: [
        {
          id: 'mem-abc-123',
          event: 'ADD',
          user_id: 'alice',
          data: { memory: 'User prefers dark mode' },
        },
      ],
    }),
  };
}
