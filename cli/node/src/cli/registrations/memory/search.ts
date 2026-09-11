import fs from 'node:fs';
import type { Command } from 'commander';
import { printError } from '../../../output/branding.js';
import { exit } from '../../../runtime/exit.js';
import { setAgentMode, stdinIsPiped } from '../../../runtime/state.js';

import { resolveIds } from '../../../application/entity-ids.js';
import { getBackendAndConfig } from '../../context.js';

/**
 * 注册 memory/search 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerMemorySearch(program: Command): void {
  /**
   * 读取全局选项并同步本次调用的代理输出模式。
   * @returns 本次调用是否采用代理输出模式。
   */
  function checkAgentMode(): boolean {
    const value = !!(program.opts().json || program.opts().agent);
    setAgentMode(value);
    return value;
  }

  program
    .command('search [query]')
    .description('Query your memory store — semantic, keyword, or hybrid retrieval.')
    .option('-u, --user-id <id>', 'Filter by user.')
    .option('--agent-id <id>', 'Filter by agent.')
    .option('--app-id <id>', 'Filter by app.')
    .option('--run-id <id>', 'Filter by run.')
    .option('-k, --top-k <n>', 'Number of results.', (v) => Number.parseInt(v), 10)
    .option('--threshold <n>', 'Minimum similarity score.', (v) => Number.parseFloat(v), 0.3)
    .option('--rerank', 'Enable reranking (Platform only).', false)
    .option('--keyword', 'Use keyword search.', false)
    .option(
      '--filter <json>',
      'Advanced filter as JSON: {"AND": [...]} or {"OR": [...]}, e.g. {"AND": [{"categories": {"in": ["work"]}}]}.'
    )
    .option('--fields <list>', 'Specific fields to return (comma-separated).')
    .option('--show-expired', 'Include expired memories.', false)
    .option(
      '--reference-date <date>',
      'Reference date for relative queries (YYYY-MM-DD or unix timestamp).'
    )
    .option('--latest-only', 'Only return the latest version of each memory.', false)
    .option('-o, --output <format>', 'Output: text, json, table.', 'text')
    .option('--api-key <key>', 'Override API key.')
    .option('--base-url <url>', 'Override API base URL.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo search "preferences" --user-id alice\n  $ memgo search "tools" -u alice -o json -k 5\n  $ echo "preferences" | memgo search -u alice\n  $ memgo search "invoices" -u alice --filter \'{"AND": [{"categories": {"in": ["work"]}}]}\''
    )
    .action(async (query, opts) => {
      let resolvedQuery = query;
      if (!resolvedQuery && stdinIsPiped()) {
        resolvedQuery = fs.readFileSync(0, 'utf-8').trim();
      }
      if (!resolvedQuery) {
        printError('No query provided. Pass a query argument or pipe via stdin.');
        exit(1);
      }
      const { cmdSearch } = await import('../../../cli/commands/memory/search.js');
      const isAgent = checkAgentMode();
      const { backend, config } = await getBackendAndConfig(opts.apiKey, opts.baseUrl);
      const ids = resolveIds(config, opts);
      const output = isAgent ? 'agent' : opts.output;
      await cmdSearch(backend, resolvedQuery, {
        ...ids,
        topK: opts.topK,
        threshold: opts.threshold,
        rerank: opts.rerank,
        keyword: opts.keyword,
        filterJson: opts.filter,
        fields: opts.fields,
        showExpired: opts.showExpired,
        referenceDate: opts.referenceDate,
        latestOnly: opts.latestOnly,
        output,
      });
    });
}
