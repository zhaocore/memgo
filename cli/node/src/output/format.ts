import type { JsonObject } from '../backend/json.js';
import { parseJsonValue } from '../backend/json.js';
/** CLI 文本、JSON、表格和静默输出。 */

import boxen from 'boxen';
import Table from 'cli-table3';
import { takeNotice } from '../runtime/state.js';
import { colors, sym } from './branding.js';

const { brand, accent, success, error: errorColor, dim } = colors;

/**
 * 提取日期部分；无法解析时保留原字符串的前十位。
 * @param dtStr - 待展示的日期字符串。
 * @returns 日期文本；空输入返回未定义。
 */
function formatDate(dtStr?: string): string | undefined {
  if (!dtStr) return undefined;
  try {
    const dt = new Date(dtStr.replace('Z', '+00:00'));
    return dt.toISOString().slice(0, 10);
  } catch {
    return dtStr?.slice(0, 10);
  }
}

/**
 * 将记忆列表输出为带摘要的终端文本。
 * @param memories - 后端返回的记忆列表。
 * @param title - 展示标题。
 */
export function formatMemoriesText(memories: JsonObject[], title: string): void {
  const count = memories.length;
  console.log(`\n${brand(`Found ${count} ${title}:`)}\n`);

  for (let i = 0; i < memories.length; i++) {
    const mem = memories[i];
    const memoryText = (mem.memory ?? mem.text ?? '') as string;
    const memId = ((mem.id as string) ?? '').slice(0, 8);
    const score = mem.score as number | undefined;
    const created = formatDate(mem.created_at as string | undefined);
    let category: string | undefined;
    const cats = mem.categories;
    if (Array.isArray(cats)) {
      category = cats[0] as string | undefined;
    }

    console.log(`  ${i + 1}. ${memoryText}`);

    const details: string[] = [];
    if (score !== undefined) details.push(`Score: ${score.toFixed(2)}`);
    if (memId) details.push(`ID: ${memId}`);
    if (created) details.push(`Created: ${created}`);
    if (category) details.push(`Category: ${category}`);

    if (details.length > 0) {
      console.log(`     ${dim(details.join(' · '))}`);
    }
    console.log();
  }
}

/**
 * 将记忆列表输出为表格。
 * @param memories - 后端返回的记忆列表。
 * @param opts - 本次操作选项。
 * @param opts.showScore - 在表格中显示匹配分数。
 */
export function formatMemoriesTable(memories: JsonObject[], opts: { showScore?: boolean }): void {
  const head = opts.showScore
    ? [accent('ID'), accent('Score'), accent('Memory'), accent('Category'), accent('Created')]
    : [accent('ID'), accent('Memory'), accent('Category'), accent('Created')];
  const colWidths = opts.showScore ? [38, 8, 40, 16, 14] : [38, 40, 16, 14];
  const table = new Table({
    head,
    colWidths,
    wordWrap: true,
    style: { head: [], border: [] },
  });

  for (const mem of memories) {
    const memId = (mem.id as string) ?? '';
    let memoryText = (mem.memory ?? mem.text ?? '') as string;
    if (memoryText.length > 60) {
      memoryText = `${memoryText.slice(0, 57)}...`;
    }
    const categories = mem.categories;
    const cat =
      Array.isArray(categories) && categories.length > 0
        ? categories.length > 1
          ? `${categories[0]} (+${categories.length - 1})`
          : (categories[0] as string)
        : '—';
    const created = formatDate(mem.created_at as string | undefined) ?? '—';
    if (opts.showScore) {
      const score = mem.score as number | undefined;
      const scoreStr = score !== undefined ? score.toFixed(2) : '—';
      table.push([dim(memId), scoreStr, memoryText, cat, created]);
    } else {
      table.push([dim(memId), memoryText, cat, created]);
    }
  }

  console.log();
  console.log(table.toString());
  console.log();
}

/**
 * 将数据序列化为缩进 JSON 并输出。
 * @param data - 需要输出的数据。
 */
export function formatJson(data: unknown): void {
  console.log(JSON.stringify(data, null, 2));
}

/**
 * 按所选格式输出单条记忆详情。
 * @param mem - 单条记忆对象。
 * @param output - 终端输出格式。
 */
export function formatSingleMemory(mem: JsonObject, output: string): void {
  if (output === 'json') {
    formatJson(mem);
    return;
  }

  const memoryText = (mem.memory ?? mem.text ?? '') as string;
  const memId = (mem.id ?? '') as string;

  const lines: string[] = [];
  lines.push(`  ${memoryText}`);
  lines.push('');

  if (memId) lines.push(`  ${dim('ID:')}         ${memId}`);
  const created = formatDate(mem.created_at as string | undefined);
  if (created) lines.push(`  ${dim('Created:')}    ${created}`);
  const updated = formatDate(mem.updated_at as string | undefined);
  if (updated) lines.push(`  ${dim('Updated:')}    ${updated}`);
  const meta = mem.metadata;
  if (meta) lines.push(`  ${dim('Metadata:')}   ${JSON.stringify(meta)}`);
  const categories = mem.categories;
  if (categories) {
    const catStr = Array.isArray(categories) ? categories.join(', ') : String(categories);
    lines.push(`  ${dim('Categories:')} ${catStr}`);
  }

  const content = lines.join('\n');
  console.log();
  console.log(
    boxen(content, {
      title: brand('Memory'),
      titleAlignment: 'left',
      borderColor: 'magenta',
      padding: 1,
    })
  );
  console.log();
}

/**
 * 展示添加结果，合并相同事件的待处理提示。
 * @param result - 添加记忆的响应对象或列表。
 * @param output - 终端输出格式。
 */
export function formatAddResult(result: JsonObject | JsonObject[], output: string): void {
  if (output === 'json') {
    formatJson(result);
    return;
  }
  if (output === 'quiet') return;

  const results: JsonObject[] = Array.isArray(result)
    ? result
    : ((result.results as JsonObject[]) ?? [result]);

  if (!results.length) {
    console.log(`  ${dim('No memories extracted.')}`);
    return;
  }

  console.log();
  const seenPendingEvents = new Set<string>();
  for (const r of results) {
    // 识别异步 PENDING 响应。
    if (r.status === 'PENDING') {
      const eventId = (r.event_id as string) ?? '';
      // 相同 event_id 的待处理项只显示一次。
      if (eventId && seenPendingEvents.has(eventId)) continue;
      if (eventId) seenPendingEvents.add(eventId);
      const icon = accent(sym('⧗', '...'));
      const parts = [`  ${icon} ${dim('Queued'.padEnd(10))}`, 'Processing in background'];
      console.log(parts.join('  '));
      if (eventId) {
        console.log(`  ${dim(`  event_id: ${eventId}`)}`);
        console.log(`  ${dim(`  → Check status: memgo event status ${eventId}`)}`);
      }
      continue;
    }

    const event = (r.event ?? 'ADD') as string;
    const memory = (r.memory ?? r.text ?? r.content ?? r.data ?? '') as string;
    const memId = ((r.id as string) ?? (r.memory_id as string) ?? '').slice(0, 8);

    let icon: string;
    let label: string;
    if (event === 'ADD') {
      icon = success('+');
      label = 'Added';
    } else if (event === 'UPDATE') {
      icon = accent('~');
      label = 'Updated';
    } else if (event === 'DELETE') {
      icon = errorColor('-');
      label = 'Deleted';
    } else if (event === 'NOOP') {
      icon = dim('·');
      label = 'No change';
    } else {
      icon = dim('?');
      label = event;
    }

    const parts = [`  ${icon} ${dim(label.padEnd(10))}`];
    if (memory) parts.push(memory);
    if (memId) parts.push(dim(`(${memId})`));
    console.log(parts.join('  '));
  }
  console.log();
}

/**
 * 构造命令 JSON 信封并附加待展示的平台通知。
 * @param opts - 本次操作选项。
 * @param opts.command - 当前命令名称。
 * @param opts.data - 需要输出的数据。
 * @param opts.durationMs - 操作耗时，单位为毫秒。
 * @param opts.scope - 本次操作的实体范围。
 * @param opts.count - 结果数量。
 * @param opts.status - 业务执行状态。
 * @param opts.error - 需要输出的错误描述。
 */
export function formatJsonEnvelope(opts: {
  command: string;
  data: unknown;
  durationMs?: number;
  scope?: Record<string, string | undefined>;
  count?: number;
  status?: string;
  error?: string;
}): void {
  const envelope: JsonObject = {
    status: opts.status ?? 'success',
    command: opts.command,
  };
  if (opts.durationMs !== undefined) envelope.duration_ms = opts.durationMs;
  if (opts.scope !== undefined) envelope.scope = opts.scope;
  if (opts.count !== undefined) envelope.count = opts.count;
  if (opts.error) envelope.error = opts.error;
  envelope.data = opts.data === undefined ? undefined : parseJsonValue(opts.data);

  // 未认领账号提示放入 JSON 信封，调用方无需读取 HTTP 头。

  const notice = takeNotice();
  if (notice) envelope.memgo_notice = notice;

  console.log(JSON.stringify(envelope, null, 2));
}

/**
 * 复制指定字段，忽略不存在的键。
 * @param obj - 字段来源对象。
 * @param keys - 需要保留的字段名。
 * @returns 只包含指定已存在字段的新对象。
 */
function pick(obj: JsonObject, keys: string[]): JsonObject {
  const result: JsonObject = {};
  for (const key of keys) {
    if (key in obj) result[key] = obj[key];
  }
  return result;
}

/**
 * 按命令保留代理输出需要的字段。
 * @param command - 当前命令名称。
 * @param data - 需要输出的数据。
 * @returns 按命令裁剪字段后的数据。
 */
export function sanitizeAgentData(command: string, data: unknown): unknown {
  if (data === null || data === undefined) return data;

  switch (command) {
    case 'add': {
      const items = Array.isArray(data) ? data : [data];
      return items.map((item) => {
        const r = item as JsonObject;
        if (r.status === 'PENDING') return pick(r, ['status', 'event_id']);
        return pick(r, ['id', 'memory', 'event']);
      });
    }
    case 'search':
      return (data as JsonObject[]).map((r) =>
        pick(r, ['id', 'memory', 'score', 'created_at', 'categories', 'expiration_date'])
      );
    case 'list':
      return (data as JsonObject[]).map((r) =>
        pick(r, ['id', 'memory', 'created_at', 'categories', 'expiration_date'])
      );
    case 'get': {
      const r = data as JsonObject;
      return pick(r, [
        'id',
        'memory',
        'created_at',
        'updated_at',
        'categories',
        'metadata',
        'expiration_date',
      ]);
    }
    case 'update': {
      const r = data as JsonObject;
      return pick(r, ['id', 'memory', 'expiration_date']);
    }
    case 'delete':
    case 'delete-all':
    case 'entity delete':
      return data;
    case 'entity list':
      return (data as JsonObject[]).map((r) => ({
        name: (r.name ?? r.id) as string,
        ...pick(r, ['type', 'count']),
      }));
    case 'event list':
      return (data as JsonObject[]).map((r) =>
        pick(r, ['id', 'event_type', 'status', 'latency', 'created_at'])
      );
    case 'event status': {
      const ev = data as JsonObject;
      const rawResults = (ev.results as JsonObject[] | undefined) ?? [];
      const sanitizedResults = rawResults.map((r) => {
        const nested = r.data as JsonObject | undefined;
        return {
          id: r.id,
          event: r.event,
          user_id: r.user_id,
          memory: nested?.memory ?? null,
        };
      });
      return {
        ...pick(ev, ['id', 'event_type', 'status', 'latency', 'created_at', 'updated_at']),
        results: sanitizedResults,
      };
    }
    default:
      return data;
  }
}

/**
 * 输出精简的代理 JSON 信封。
 * @param opts - 本次操作选项。
 * @param opts.command - 当前命令名称。
 * @param opts.data - 需要输出的数据。
 * @param opts.durationMs - 操作耗时，单位为毫秒。
 * @param opts.scope - 本次操作的实体范围。
 * @param opts.count - 结果数量。
 */
export function formatAgentEnvelope(opts: {
  command: string;
  data: unknown;
  durationMs?: number;
  scope?: Record<string, string | undefined>;
  count?: number;
}): void {
  const envelope: JsonObject = {
    status: 'success',
    command: opts.command,
  };
  if (opts.durationMs !== undefined) envelope.duration_ms = opts.durationMs;
  if (opts.scope) {
    const filtered = Object.fromEntries(Object.entries(opts.scope).filter(([, v]) => v));
    if (Object.keys(filtered).length > 0) envelope.scope = filtered;
  }
  if (opts.count !== undefined) envelope.count = opts.count;
  envelope.data = parseJsonValue(sanitizeAgentData(opts.command, opts.data));

  // 在 JSON 信封中显示未认领账号提示。

  const notice = takeNotice();
  if (notice) envelope.memgo_notice = notice;

  console.log(JSON.stringify(envelope, null, 2));
}

/**
 * 展示结果数量、分页、实体范围及耗时摘要。
 * @param opts - 本次操作选项。
 * @param opts.count - 结果数量。
 * @param opts.durationSecs - 操作耗时，单位为秒。
 * @param opts.page - 分页页码。
 * @param opts.scopeIds - 用于摘要的实体范围。
 */
export function printResultSummary(opts: {
  count: number;
  durationSecs?: number;
  page?: number;
  scopeIds?: Record<string, string | undefined>;
}): void {
  const parts = [`${opts.count} result${opts.count !== 1 ? 's' : ''}`];
  if (opts.page !== undefined) parts.push(`page ${opts.page}`);
  if (opts.scopeIds) {
    const scopeParts = Object.entries(opts.scopeIds)
      .filter(([, v]) => v)
      .map(([k, v]) => `${k}=${v}`);
    if (scopeParts.length > 0) parts.push(scopeParts.join(', '));
  }
  if (opts.durationSecs !== undefined) parts.push(`${opts.durationSecs.toFixed(2)}s`);

  console.log(`  ${dim(parts.join(' · '))}`);
  console.log();
}
