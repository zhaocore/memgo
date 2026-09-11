import { APIError } from './errors.js';
import { type HttpRequest, readResponse, requestOnce, requestRead } from './http.js';
import type { JsonObject } from './json.js';
import { jsonObject, jsonRecords, jsonResult, parseJsonValue } from './json.js';
/** 由调用层提供的输出上下文，协议适配器不依赖 CLI 全局状态。 */
export interface PlatformContext {
  callerType(): 'agent' | 'user';
  notice(message: string | null | undefined): void;
}
/** 平台后端负责 api.memgo.ai 的协议映射。 */

import type { PlatformConfig } from '../config/types.js';

import { CLI_VERSION } from '../version.js';
import type {
  AddOptions,
  Backend,
  DeleteOptions,
  EntityIds,
  ListOptions,
  SearchOptions,
  UpdateOptions,
} from './types.js';

/**
 * 将标识符编码为单个 URL 路径片段。
 * @param value - 待处理的字段值。
 * @returns 可安全拼接到路径中的编码字符串。
 */
function encodePathSegment(value: unknown): string {
  return encodeURIComponent(String(value));
}

export class PlatformBackend implements Backend {
  private baseUrl: string;
  private headers: Record<string, string>;

  /**
   * 保存连接配置并构造当前后端的鉴权请求头。
   * @param config - 基础地址与鉴权配置。
   * @param context - 由调用层注入的身份与通知依赖。
   */
  constructor(
    config: PlatformConfig,
    private readonly context: PlatformContext
  ) {
    this.baseUrl = config.baseUrl.replace(/\/+$/, '');
    this.headers = {
      Authorization: `Token ${config.apiKey}`,
      'Content-Type': 'application/json',
      'X-MemGo-Source': 'cli',
      'X-MemGo-Client-Language': 'node',
      'X-MemGo-Client-Version': CLI_VERSION,
    };
  }

  /**
   * 装配请求与鉴权头，执行网络请求并解析响应。
   * @param method - HTTP 方法。
   * @param path - 相对于后端基础地址的接口路径。
   * @param opts - 本次操作参数。
   * @param opts.json - 待序列化的请求对象。
   * @param opts.params - URL 查询参数。
   * @param opts.timeoutMs - 单次请求超时，单位为毫秒。
   * @returns 解析后的响应数据，交由操作方法继续校验。
   */
  private async _request(
    method: string,
    path: string,
    opts?: {
      json?: unknown;
      params?: Record<string, string>;
      timeoutMs?: number;
    }
  ): Promise<unknown> {
    const query = opts?.params ? `?${new URLSearchParams(opts.params)}` : '';
    const request: HttpRequest = {
      url: `${this.baseUrl}${path}${query}`,
      path,
      method,
      headers: {
        ...this.headers,
        'X-MemGo-Caller-Type': this.context.callerType(),
      },
      body: opts?.json === undefined ? undefined : JSON.stringify(opts.json),
      timeoutMs: opts?.timeoutMs ?? 30000,
    };
    const response = await (method === 'GET' ? requestRead(request) : requestOnce(request));
    const data = parseJsonValue(await readResponse(response, request));

    if (data && typeof data === 'object' && !Array.isArray(data) && 'memgo_notice' in data) {
      const { memgo_notice, ...rest } = data;
      if (typeof memgo_notice !== 'string')
        throw new APIError(path, 'memgo_notice must be a string');
      this.context.notice(memgo_notice);
      return rest;
    }
    if (
      Array.isArray(data) &&
      data[0] &&
      typeof data[0] === 'object' &&
      'memgo_notice' in data[0]
    ) {
      const { memgo_notice, ...first } = data[0];
      if (typeof memgo_notice !== 'string')
        throw new APIError(path, 'memgo_notice must be a string');
      this.context.notice(memgo_notice);
      return [first, ...data.slice(1)];
    }
    this.context.notice(response.headers.get('X-MemGo-Notice-Message'));

    return data;
  }

  /**
   * 将记忆输入映射为后端添加请求。
   * @param content - 直接提供的记忆文本。
   * @param messages - 结构化对话消息。
   * @param opts - 本次操作参数。
   * @returns 后端添加响应，保留对象或数组形状。
   */
  async add(
    content: string | undefined,
    messages: JsonObject[] | undefined,
    opts: AddOptions
  ): Promise<JsonObject | JsonObject[]> {
    const payload: JsonObject = {};

    if (messages) {
      payload.messages = messages;
    } else if (content) {
      payload.messages = [{ role: 'user', content }];
    }

    if (opts.userId) payload.user_id = opts.userId;
    if (opts.agentId) payload.agent_id = opts.agentId;
    if (opts.appId) payload.app_id = opts.appId;
    if (opts.runId) payload.run_id = opts.runId;
    if (opts.metadata) payload.metadata = opts.metadata;
    if (opts.immutable) payload.immutable = true;
    if (opts.infer === false) payload.infer = false;
    if (opts.expires) payload.expiration_date = opts.expires;
    if (opts.customInstructions) payload.custom_instructions = opts.customInstructions;
    if (opts.agentCustomInstructions)
      payload.agent_custom_instructions = opts.agentCustomInstructions;
    if (opts.customCategories) payload.custom_categories = opts.customCategories;
    if (opts.structuredDataSchema) payload.structured_data_schema = opts.structuredDataSchema;
    if (opts.timestamp !== undefined) payload.timestamp = opts.timestamp;
    payload.source = 'CLI';

    return jsonResult(
      await this._request('POST', '/v3/memories/add/', {
        json: payload,
      })
    );
  }

  /**
   * 合并实体条件与显式布尔筛选表达式。
   * @param opts - 本次操作参数。
   * @param opts.userId - 用户 ID。
   * @param opts.agentId - 代理 ID。
   * @param opts.appId - 应用 ID。
   * @param opts.runId - 会话 ID。
   * @param opts.extraFilters - 调用方提供的额外筛选表达式。
   * @returns 合并后的筛选对象；没有条件时返回未定义。
   */
  private _buildFilters(opts: {
    userId?: string;
    agentId?: string;
    appId?: string;
    runId?: string;
    extraFilters?: JsonObject;
  }): JsonObject | undefined {
    // 调用方已构造 AND/OR 过滤器时保持原结构。
    if (opts.extraFilters && ('AND' in opts.extraFilters || 'OR' in opts.extraFilters)) {
      return opts.extraFilters;
    }

    const andConditions: JsonObject[] = [];
    if (opts.userId) andConditions.push({ user_id: opts.userId });
    if (opts.agentId) andConditions.push({ agent_id: opts.agentId });
    if (opts.appId) andConditions.push({ app_id: opts.appId });
    if (opts.runId) andConditions.push({ run_id: opts.runId });

    if (opts.extraFilters) {
      for (const [k, v] of Object.entries(opts.extraFilters)) {
        andConditions.push({ [k]: v });
      }
    }

    if (andConditions.length === 1) return andConditions[0];
    if (andConditions.length > 1) return { AND: andConditions };
    return undefined;
  }

  /**
   * 将实体范围与搜索条件映射为后端查询。
   * @param query - 记忆查询文本。
   * @param opts - 本次操作参数。
   * @returns 匹配的记忆列表。
   */
  async search(query: string, opts: SearchOptions): Promise<JsonObject[]> {
    const payload: JsonObject = {
      query,
      top_k: opts.topK ?? 10,
      threshold: opts.threshold ?? 0.3,
    };

    const apiFilters = this._buildFilters({
      userId: opts.userId,
      agentId: opts.agentId,
      appId: opts.appId,
      runId: opts.runId,
      extraFilters: opts.filters,
    });
    if (apiFilters) payload.filters = apiFilters;
    if (opts.rerank) payload.rerank = true;
    if (opts.keyword) payload.keyword_search = true;
    if (opts.fields) payload.fields = opts.fields;
    if (opts.showExpired) payload.show_expired = true;
    if (opts.referenceDate !== undefined) payload.reference_date = opts.referenceDate;
    if (opts.latestOnly) payload.latest_only = true;
    payload.source = 'CLI';

    const result = (await this._request('POST', '/v3/memories/search/', {
      json: payload,
    })) as unknown;
    return jsonRecords(result);
  }

  /**
   * 按记忆 ID 读取详情并校验响应对象。
   * @param memoryId - 目标记忆 ID。
   * @returns 指定记忆的详情对象。
   */
  async get(memoryId: string): Promise<JsonObject> {
    return jsonObject(
      await this._request('GET', `/v1/memories/${encodePathSegment(memoryId)}/`, {
        params: { source: 'CLI' },
      })
    );
  }

  /**
   * 按筛选与分页条件读取记忆列表。
   * @param opts - 本次操作参数。
   * @returns 当前页的记忆列表。
   */
  async listMemories(opts: ListOptions): Promise<JsonObject[]> {
    const payload: JsonObject = {};
    const params: Record<string, string> = {
      page: String(opts.page ?? 1),
      page_size: String(opts.pageSize ?? 100),
    };

    const extra: JsonObject = {};
    if (opts.category) {
      extra.categories = { contains: opts.category };
    }
    if (opts.after) {
      extra.created_at = {
        ...(extra.created_at as JsonObject | undefined),
        gte: opts.after,
      };
    }
    if (opts.before) {
      extra.created_at = {
        ...(extra.created_at as JsonObject | undefined),
        lte: opts.before,
      };
    }

    const apiFilters = this._buildFilters({
      userId: opts.userId,
      agentId: opts.agentId,
      appId: opts.appId,
      runId: opts.runId,
      extraFilters: Object.keys(extra).length > 0 ? extra : undefined,
    });
    if (apiFilters) payload.filters = apiFilters;
    if (opts.showExpired) payload.show_expired = true;
    if (opts.latestOnly) payload.latest_only = true;
    payload.source = 'CLI';

    const result = (await this._request('POST', '/v3/memories/', {
      json: payload,
      params,
    })) as unknown;
    return jsonRecords(result);
  }

  /**
   * 映射部分更新字段并提交记忆修改。
   * @param memoryId - 目标记忆 ID。
   * @param content - 直接提供的记忆文本。
   * @param metadata - 需要更新的元数据对象。
   * @param opts - 本次操作参数。
   * @returns 后端更新响应。
   */
  async update(
    memoryId: string,
    content: string | undefined,
    metadata: JsonObject | undefined,
    opts: UpdateOptions
  ): Promise<JsonObject> {
    const payload: JsonObject = {};
    if (content) payload.text = content;
    if (metadata) payload.metadata = metadata;
    if (opts.expirationDate) payload.expiration_date = opts.expirationDate;
    if (opts.timestamp !== undefined) payload.timestamp = opts.timestamp;
    payload.source = 'CLI';
    return jsonObject(
      await this._request('PUT', `/v1/memories/${encodePathSegment(memoryId)}/`, {
        json: payload,
      })
    );
  }

  /**
   * 按单条或实体范围提交删除请求。
   * @param memoryId - 目标记忆 ID。
   * @param opts - 本次操作参数。
   * @returns 后端删除响应。
   */
  async delete(memoryId: string | undefined, opts: DeleteOptions): Promise<JsonObject> {
    if (opts.all) {
      const params: Record<string, string> = { source: 'CLI' };
      if (opts.userId) params.user_id = opts.userId;
      if (opts.agentId) params.agent_id = opts.agentId;
      if (opts.appId) params.app_id = opts.appId;
      if (opts.runId) params.run_id = opts.runId;
      return jsonObject(
        await this._request('DELETE', '/v1/memories/', {
          params,
        })
      );
    }
    if (memoryId) {
      const params: Record<string, string> = { source: 'CLI' };
      if (opts.deleteLinked) params.delete_linked = 'true';
      return jsonObject(
        await this._request('DELETE', `/v1/memories/${encodePathSegment(memoryId)}/`, { params })
      );
    }
    throw new Error('Either memoryId or --all is required');
  }

  /**
   * 逐项删除指定实体及其关联记忆。
   * @param opts - 本次操作参数。
   * @returns 各实体删除的聚合结果。
   */
  async deleteEntities(opts: EntityIds): Promise<JsonObject> {
    // 实体删除使用 v2 路径协议。
    const typeMap: [string, string | undefined][] = [
      ['user', opts.userId],
      ['agent', opts.agentId],
      ['app', opts.appId],
      ['run', opts.runId],
    ];
    const entities = typeMap.filter(([, v]) => v) as [string, string][];
    if (entities.length === 0) {
      throw new Error('At least one entity ID is required for deleteEntities.');
    }
    // 按实体类型保存每次删除结果，避免多实体删除丢失先前结果。

    const results: JsonObject = {};
    for (const [entityType, entityId] of entities) {
      results[entityType] = jsonObject(
        await this._request(
          'DELETE',
          `/v2/entities/${encodePathSegment(entityType)}/${encodePathSegment(entityId)}/`,
          { params: { source: 'CLI' } }
        )
      );
    }
    return results;
  }

  /**
   * 通过后端探活接口检查连通性。
   * @returns 后端探活响应。
   */
  async ping(): Promise<JsonObject> {
    return jsonObject(
      await this._request('GET', '/v1/ping/', {
        timeoutMs: 2400,
      })
    );
  }

  /**
   * 探测连通性并返回包含失败原因的状态对象。
   * @param _opts - 为保持 Backend 接口一致而保留的实体选项；连通性探测不使用。
   * @param _opts.userId - 用户 ID。
   * @param _opts.agentId - 代理 ID。
   * @returns 连通性、后端类型、地址及可能的错误信息。
   */
  async status(_opts: {
    userId?: string;
    agentId?: string;
  }): Promise<JsonObject> {
    try {
      await this.ping();
      return { connected: true, backend: 'platform', base_url: this.baseUrl };
    } catch (e) {
      return {
        connected: false,
        backend: 'platform',
        error: e instanceof Error ? e.message : String(e),
      };
    }
  }

  /**
   * 读取并筛选指定类型的实体。
   * @param entityType - 需要返回的实体类型。
   * @returns 匹配类型的实体列表。
   */
  async entities(entityType: string): Promise<JsonObject[]> {
    const result = (await this._request('GET', '/v1/entities/')) as unknown;
    let items = jsonRecords(result);

    const typeMap: Record<string, string> = {
      users: 'user',
      agents: 'agent',
      apps: 'app',
      runs: 'run',
    };
    const targetType = typeMap[entityType];
    if (targetType) {
      items = items.filter((e) => (e.type as string | undefined)?.toLowerCase() === targetType);
    }
    return items;
  }

  /**
   * 读取异步事件列表。
   * @returns 异步事件列表。
   */
  async listEvents(): Promise<JsonObject[]> {
    const result = (await this._request('GET', '/v1/events/')) as unknown;
    return jsonRecords(result);
  }

  /**
   * 读取指定异步事件详情。
   * @param eventId - 异步事件 ID。
   * @returns 指定事件的详情对象。
   */
  async getEvent(eventId: string): Promise<JsonObject> {
    return jsonObject(await this._request('GET', `/v1/event/${encodePathSegment(eventId)}/`));
  }
}
