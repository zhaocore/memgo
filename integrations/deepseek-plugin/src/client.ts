/**
 * MemGo OSS server 的内置 HTTP 客户端。
 *
 * 替代原托管 SDK: 直连 MemGo 自托管 server 的 OSS 面(契约见
 * integrations/README.md), 基于 Node 原生 fetch, 无任何外部运行时依赖。
 *
 * 与托管平台 SDK 的差异:
 *   - 鉴权头为 `X-API-Key`(不是 `Authorization: Token`)
 *   - 写入与搜索均为同步返回 `{results:[...]}`, 无事件轮询
 *   - 实体维度仅 user_id / agent_id / run_id, 无 app_id / custom_categories
 *   - 错误按 HTTP 状态码分类为粗粒度 ErrorKind(见 ApiError)
 */

export const DEFAULT_HOST = "https://memgo.wxget.com";

/** 范围参数: user_id 必有, agent_id / run_id 可选(snake_case, 原样发给 server)。 */
export interface Scope {
  user_id: string;
  agent_id?: string;
  run_id?: string;
}

/** 粗粒度错误分类, 与遥测 errorKind 对齐。 */
export type ErrorKind =
  | "auth"
  | "rate-limited"
  | "bad-request"
  | "server-error"
  | "network"
  | "timeout"
  | "other";

/** OSS 返回的单条记忆记录。字段为 snake_case, 原样透传。 */
export interface MemoryRecord {
  id: string;
  memory?: string;
  user_id?: string;
  agent_id?: string;
  run_id?: string;
  metadata?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

/** 客户端配置。apiKey 与 userId 均在 apply 阶段完成必填校验。 */
export interface MemoryClientConfig {
  apiKey: string;
  userId: string;
  host?: string;
}

function kindForStatus(status: number): ErrorKind {
  if (status === 401 || status === 403) return "auth";
  if (status === 429) return "rate-limited";
  if (status >= 400 && status < 500) return "bad-request";
  if (status >= 500 && status < 600) return "server-error";
  return "other";
}

/** 带 HTTP 状态码与粗粒度错误分类的 API 错误, 供 errorKind 直接读取。 */
export class ApiError extends Error {
  readonly status: number | undefined;
  readonly kind: ErrorKind;

  constructor(message: string, kind: ErrorKind, status?: number) {
    super(message);
    this.name = "ApiError";
    this.kind = kind;
    this.status = status;
  }
}

export class MemoryClient {
  private readonly apiKey: string;
  private readonly userId: string;
  private readonly host: string;

  constructor(config: MemoryClientConfig) {
    this.apiKey = config.apiKey;
    this.userId = config.userId;
    // host 缺省读 MEMGO_BASE_URL, 再回退到默认地址; 去掉结尾斜杠便于拼路径。
    this.host = (config.host ?? process.env.MEMGO_BASE_URL ?? DEFAULT_HOST).replace(/\/+$/, "");
  }

  private async request<T>(path: string, body: unknown): Promise<T> {
    let res: Response;
    try {
      res = await fetch(`${this.host}${path}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": this.apiKey,
        },
        body: JSON.stringify(body),
      });
    } catch (err) {
      // fetch 本身抛错(fetch failed / ECONNREFUSED 等)归类为网络错误。
      throw new ApiError(
        `无法连接 MemGo server ${this.host}: ${err instanceof Error ? err.message : String(err)}`,
        "network",
      );
    }
    if (!res.ok) {
      // 只携带状态码与原因短语, 不把响应体拼进消息, 避免泄露记忆内容。
      throw new ApiError(
        `MemGo API 请求失败: ${res.status} ${res.statusText}`,
        kindForStatus(res.status),
        res.status,
      );
    }
    return (await res.json()) as T;
  }

  /** 写入记忆(同步)。server 完成提取后返回记忆记录列表。 */
  async add(text: string, scope: Scope): Promise<MemoryRecord[]> {
    const body = {
      messages: [{ role: "user", content: text }],
      user_id: scope.user_id ?? this.userId,
      ...(scope.agent_id ? { agent_id: scope.agent_id } : {}),
      ...(scope.run_id ? { run_id: scope.run_id } : {}),
    };
    const data = await this.request<{ results?: MemoryRecord[] }>("/memories", body);
    return data.results ?? [];
  }

  /** 搜索记忆(同步), 返回相关记忆记录列表。 */
  async search(query: string, filters: Scope, topK: number): Promise<MemoryRecord[]> {
    const body = {
      query,
      filters: {
        user_id: filters.user_id ?? this.userId,
        ...(filters.agent_id ? { agent_id: filters.agent_id } : {}),
        ...(filters.run_id ? { run_id: filters.run_id } : {}),
      },
      top_k: topK,
    };
    const data = await this.request<{ results?: MemoryRecord[] }>("/search", body);
    return data.results ?? [];
  }
}
