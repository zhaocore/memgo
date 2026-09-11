import { run } from '../helpers/cli.js';
/** 文档中 v3 添加、搜索和列表字段须有对应 CLI 入口或明确缺口。 */

import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const OPENAPI_PATH = path.join(__dirname, '..', '..', '..', '..', 'docs', 'openapi.json');

const KNOWN_UNSURFACED: Record<string, Record<string, string>> = {
  '/v3/memories/add/': {
    includes: 'extraction hint, no CLI flag yet',
    excludes: 'extraction hint, no CLI flag yet',
    enable_graph: 'graph memory toggle, no CLI flag yet',
    output_format: 'response envelope is pinned by the CLI',
    prompt_profile_id: 'no CLI flag yet',
    temporal_reasoning: 'no CLI flag yet',
    timezone: 'no CLI flag yet',
    observation_datetime: 'no CLI flag yet, --timestamp backdates instead',
    observation_date: 'no CLI flag yet, --timestamp backdates instead',
  },
  '/v3/memories/search/': {
    categories: 'expressible through --filter',
    metadata: 'expressible through --filter',
  },
  '/v3/memories/': {
    start_date: 'covered by --after via filters.created_at.gte',
    end_date: 'covered by --before via filters.created_at.lte',
    categories: 'covered by --category via filters.categories',
    fields: 'no CLI flag yet',
    keywords: 'no CLI flag yet',
  },
};

const ADD_MAPPING: Record<string, string[]> = {
  messages: ['--messages', '--file', 'text'],
  user_id: ['--user-id'],
  agent_id: ['--agent-id'],
  app_id: ['--app-id'],
  run_id: ['--run-id'],
  metadata: ['--metadata'],
  expiration_date: ['--expires'],
  custom_instructions: ['--custom-instructions'],
  agent_custom_instructions: ['--agent-custom-instructions'],
  custom_categories: ['--custom-categories'],
  infer: ['--no-infer'],
  immutable: ['--immutable'],
  structured_data_schema: ['--structured-data-schema'],
  timestamp: ['--timestamp'],
};

const SEARCH_MAPPING: Record<string, string[]> = {
  query: ['query'],
  filters: ['--filter', '--user-id', '--agent-id', '--run-id'],
  show_expired: ['--show-expired'],
  top_k: ['--top-k'],
  threshold: ['--threshold'],
  rerank: ['--rerank'],
  reference_date: ['--reference-date'],
  fields: ['--fields'],
};

const LIST_MAPPING: Record<string, string[]> = {
  filters: ['--user-id', '--agent-id', '--run-id', '--category', '--after', '--before'],
  show_expired: ['--show-expired'],
  page: ['--page'],
  page_size: ['--page-size'],
};

function documentedFields(endpoint: string): string[] {
  const spec = JSON.parse(fs.readFileSync(OPENAPI_PATH, 'utf-8'));
  const schema = spec.paths[endpoint].post.requestBody.content['application/json'].schema;
  return Object.keys(schema.properties);
}

function helpText(command: string): string {
  const result = run([command, '--help'], {});
  if (result.exitCode !== 0) throw new Error(result.stderr);
  return result.stdout;
}

function assertAllReachable(endpoint: string, mapping: Record<string, string[]>, command: string) {
  const documented = documentedFields(endpoint);
  const help = helpText(command);
  for (const field of documented) {
    if (KNOWN_UNSURFACED[endpoint]?.[field]) continue;
    const candidates = mapping[field];
    expect(
      candidates,
      `${endpoint}: documented field "${field}" has no mapping entry for command "${command}"`
    ).toBeDefined();
    const reachable = candidates.some((flag) =>
      flag.startsWith('--') ? help.includes(flag) : true
    );
    expect(
      reachable,
      `${endpoint}: documented field "${field}" not reachable via any of ${JSON.stringify(candidates)} on command "${command}"`
    ).toBe(true);
  }
}

describe('Option parity: Node CLI reachability of documented v3 params', () => {
  it('add covers documented fields', () => {
    assertAllReachable('/v3/memories/add/', ADD_MAPPING, 'add');
  });

  it('search covers documented fields', () => {
    assertAllReachable('/v3/memories/search/', SEARCH_MAPPING, 'search');
  });

  it('list covers documented fields', () => {
    assertAllReachable('/v3/memories/', LIST_MAPPING, 'list');
  });
});
