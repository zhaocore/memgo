/**
 * MemGo 结果的紧凑、省 token 渲染, 供模型上下文阅读。
 *
 * 每个记忆一行: 分类 + 记忆文本 + 年龄 + [memgo:id]。直接倾倒原始 search
 * 信封会浪费大部分 token 在 JSON 结构上。OSS 面无 custom_categories, 无分类
 * 时省略分类前缀。
 */

export interface MemoryLike {
  id: string;
  memory?: string;
  categories?: string[];
  createdAt?: Date | string;
  /** OSS 原样返回的 snake_case 时间字段。 */
  created_at?: Date | string;
  metadata?: Record<string, unknown>;
}

export function formatAge(date: Date | string): string {
  const d = typeof date === "string" ? new Date(date) : date;
  const ms = Date.now() - d.getTime();
  const minutes = Math.floor(ms / 60_000);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function formatMemoryCompact(mem: MemoryLike): string {
  const cat = mem.categories?.[0];
  const catTag = cat ? `[${cat}] ` : "";
  const createdAt = mem.createdAt ?? mem.created_at;
  const age = createdAt ? ` (${formatAge(createdAt)})` : "";
  return `${catTag}${mem.memory ?? "(empty)"}${age} [memgo:${mem.id}]`;
}

export function formatMemoryList(memories: MemoryLike[]): string {
  if (memories.length === 0) return "No memories found.";
  return memories
    .map((m, i) => `${i + 1}. ${formatMemoryCompact(m)}`)
    .join("\n");
}

/**
 * 写入确认。
 *
 * OSS 面同步写入: client.add 已从响应解包出 results, 记录即提取结果, 直接
 * 列表渲染; 空结果给一句确认。托管平台时代的异步 PENDING / event_id 状态机
 * 分支已删除(写入即完成, 无事件轮询)。
 */
export function formatAddResult(result: unknown): string {
  const items: MemoryLike[] = Array.isArray(result)
    ? (result as MemoryLike[])
    : ((result as { results?: MemoryLike[] } | null)?.results ?? []);
  if (items.length === 0) return "Memory stored.";
  const noun = items.length === 1 ? "memory" : "memories";
  return `Stored ${items.length} ${noun}:\n${formatMemoryList(items)}`;
}
