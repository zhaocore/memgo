import fs from 'node:fs';
import { _validateExpires } from '../../../application/memory/expiration.js';
import { parseCategories, parseMessages, parseObject } from '../../../application/memory/input.js';
import type { JsonObject } from '../../../backend/json.js';
import type { Backend } from '../../../backend/types.js';
import { printError, printScope, printSuccess, timedStatus } from '../../../output/branding.js';
import { formatAddResult, formatAgentEnvelope } from '../../../output/format.js';
import { exit } from '../../../runtime/exit.js';
import { setCurrentCommand, stdinIsPiped } from '../../../runtime/state.js';

/**
 * 解析文本、文件或消息输入，添加记忆并输出结果。
 * @param backend - 已装配的后端连接器。
 * @param text - 待解析或提交的文本。
 * @param opts - 本次操作选项。
 * @param opts.userId - 用户 ID。
 * @param opts.agentId - 代理 ID。
 * @param opts.appId - 应用 ID。
 * @param opts.runId - 会话 ID。
 * @param opts.messages - 消息列表的 JSON 文本。
 * @param opts.file - 记忆输入文件路径。
 * @param opts.metadata - 记忆元数据的 JSON 文本。
 * @param opts.immutable - 记忆不可变选项。
 * @param opts.infer - 是否启用记忆提取。
 * @param opts.expires - 记忆到期日期，格式为 YYYY-MM-DD。
 * @param opts.categories - 记忆分类筛选或标记。
 * @param opts.customInstructions - 记忆提取的自定义指令。
 * @param opts.agentCustomInstructions - 代理记忆的自定义指令。
 * @param opts.customCategories - 自定义分类描述的 JSON 文本。
 * @param opts.structuredDataSchema - 结构化提取模式的 JSON 文本。
 * @param opts.timestamp - 业务时间戳。
 * @param opts.output - 终端输出格式。
 */
export async function cmdAdd(
  backend: Backend,
  text: string | undefined,
  opts: {
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
    messages?: string;
    file?: string;
    metadata?: string;
    immutable: boolean;
    infer?: boolean;
    expires?: string;
    categories?: string;
    customInstructions?: string;
    agentCustomInstructions?: string;
    customCategories?: string;
    structuredDataSchema?: string;
    timestamp?: number;
    output: string;
  }
): Promise<void> {
  setCurrentCommand('add');

  if (opts.categories) {
    printError('--categories is not supported on add. Use --custom-categories instead.');
    exit(1);
  }

  let msgs: JsonObject[] | undefined;
  let content = text;

  // 从文件读取输入。
  if (opts.file) {
    try {
      const raw = fs.readFileSync(opts.file, 'utf-8');
      msgs = parseMessages(raw);
    } catch (e) {
      printError(`Failed to read file: ${e instanceof Error ? e.message : e}`);
      exit(1);
    }
  }
  // 解析消息 JSON。
  else if (opts.messages) {
    try {
      msgs = parseMessages(opts.messages);
    } catch (e) {
      printError(`Invalid JSON in --messages: ${e instanceof Error ? e.message : e}`);
      exit(1);
    }
  }
  // 只从实际管道或重定向文件读取标准输入。
  else if (!content && stdinIsPiped()) {
    content = fs.readFileSync(0, 'utf-8').trim();
  }

  if (content !== undefined && content.trim() === '') {
    printError('Content cannot be empty.');
    exit(1);
  }
  if (!content && !msgs) {
    printError('No content provided. Pass text, --messages, --file, or pipe via stdin.');
    exit(1);
  }

  let meta: JsonObject | undefined;
  if (opts.metadata) {
    try {
      meta = parseObject(opts.metadata);
    } catch {
      printError('Invalid JSON in --metadata.');
      exit(1);
    }
  }

  let customCats: Record<string, string>[] | undefined;
  if (opts.customCategories) {
    try {
      customCats = parseCategories(opts.customCategories);
    } catch {
      printError('Invalid JSON in --custom-categories.');
      exit(1);
    }
  }

  let schema: JsonObject | undefined;
  if (opts.structuredDataSchema) {
    try {
      schema = parseObject(opts.structuredDataSchema);
    } catch {
      printError('Invalid JSON in --structured-data-schema.');
      exit(1);
    }
  }

  if (opts.expires) _validateExpires(opts.expires);

  let result: JsonObject | JsonObject[];
  try {
    result = await timedStatus('Adding memory...', async () => {
      return backend.add(content ?? undefined, msgs, {
        userId: opts.userId,
        agentId: opts.agentId,
        appId: opts.appId,
        runId: opts.runId,
        metadata: meta,
        immutable: opts.immutable,
        infer: opts.infer !== false,
        expires: opts.expires,
        customInstructions: opts.customInstructions,
        agentCustomInstructions: opts.agentCustomInstructions,
        customCategories: customCats,
        structuredDataSchema: schema,
        timestamp: opts.timestamp,
      });
    });
  } catch (e) {
    printError(e instanceof Error ? e.message : String(e));
    exit(1);
  }

  if (opts.output === 'quiet') return;

  // 所有输出模式按 event_id 去重待处理项。
  const rawResults: JsonObject[] = Array.isArray(result)
    ? result
    : ((result.results as JsonObject[]) ?? [result]);
  const seenEvents = new Set<string>();
  const deduped: JsonObject[] = [];
  for (const r of rawResults) {
    if (r.status === 'PENDING') {
      const eid = (r.event_id as string) ?? '';
      if (eid && seenEvents.has(eid)) continue;
      if (eid) seenEvents.add(eid);
    }
    deduped.push(r);
  }
  // 将去重结果传给输出模块。
  const dedupedResult: JsonObject | JsonObject[] = Array.isArray(result)
    ? deduped
    : { ...result, results: deduped };

  if (opts.output === 'agent') {
    const scope: Record<string, string | undefined> = {
      user_id: opts.userId,
      agent_id: opts.agentId,
      app_id: opts.appId,
      run_id: opts.runId,
    };
    formatAgentEnvelope({
      command: 'add',
      data: deduped,
      scope,
      count: deduped.length,
    });
    return;
  }

  if (opts.output === 'json') {
    formatAddResult(dedupedResult, opts.output);
    return;
  }

  console.log();
  printScope({
    user_id: opts.userId,
    agent_id: opts.agentId,
    app_id: opts.appId,
    run_id: opts.runId,
  });
  const count = deduped.length;
  const allPending = count > 0 && deduped.every((r) => r.status === 'PENDING');
  if (allPending) {
    printSuccess(`Memory queued — ${count} event${count !== 1 ? 's' : ''} pending`);
  } else {
    printSuccess(`Memory processed — ${count} memor${count === 1 ? 'y' : 'ies'} extracted`);
  }
  formatAddResult(dedupedResult, opts.output);
}
