import { APIError } from './errors.js';
import { type HttpRequest, readResponse, requestOnce, requestRead } from './http.js';
import type { JsonObject } from './json.js';
import { jsonObject, jsonRecords, jsonResult, parseJsonValue } from './json.js';
/** 自托管后端遵守 doc-02 和 X-API-Key 鉴权；平台专属选项明确报错。 */

import type { PlatformConfig } from '../config/types.js';
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

export class OSSBackend implements Backend {
  private baseUrl: string;
  private headers: Record<string, string>;

  /**
   * 保存连接配置并构造当前后端的鉴权请求头。
   * @param config - 基础地址与鉴权配置。
   */
  constructor(config: PlatformConfig) {
    this.baseUrl = config.baseUrl.replace(/\/+$/, '');
    this.headers = {
      'X-API-Key': config.apiKey,
      'Content-Type': 'application/json',
    };
  }

  // ponytail: 600s 超时对齐 Go OSSBackend —— infer=true 走真实 LLM 时远超普通 API 延迟

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
      headers: this.headers,
      body: opts?.json === undefined ? undefined : JSON.stringify(opts.json),
      timeoutMs: opts?.timeoutMs ?? 600000,
    };
    const response = await (method === 'GET' ? requestRead(request) : requestOnce(request));
    const data = parseJsonValue(await readResponse(response, request));

    return data;
  }

  /**
   * 对 OSS 不支持的操作抛出明确的接口错误。
   * @param message - 具体失败原因。
   */
  private _reject(message: string): never {
    throw new APIError('oss-backend', message);
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
    if (opts.appId) {
      this._reject('app_id is not supported on the OSS backend');
    }
    if (opts.immutable) {
      this._reject('--immutable is not supported on the OSS backend');
    }
    if (opts.structuredDataSchema || opts.customCategories || opts.agentCustomInstructions) {
      this._reject('platform-only extraction options are not supported on the OSS backend');
    }

    const payload: JsonObject = {};
    if (messages) {
      payload.messages = messages;
    } else if (content) {
      payload.messages = [{ role: 'user', content }];
    }
    if (opts.userId) payload.user_id = opts.userId;
    if (opts.agentId) payload.agent_id = opts.agentId;
    if (opts.runId) payload.run_id = opts.runId;
    if (opts.metadata) payload.metadata = opts.metadata;
    if (opts.infer === false) payload.infer = false;
    if (opts.expires) payload.expiration_date = opts.expires;
    if (opts.customInstructions) payload.prompt = opts.customInstructions;

    return jsonResult(
      await this._request('POST', '/memories', {
        json: payload,
      })
    );
  }

  /**
   * 将实体范围与搜索条件映射为后端查询。
   * @param query - 记忆查询文本。
   * @param opts - 本次操作参数。
   * @returns 匹配的记忆列表。
   */
  async search(query: string, opts: SearchOptions): Promise<JsonObject[]> {
    if (opts.keyword) {
      this._reject('--keyword is not supported on the OSS backend');
    }
    if (opts.referenceDate !== undefined || opts.latestOnly) {
      this._reject('--reference-date/--latest-only are not supported on the OSS backend');
    }
    if (opts.appId) {
      this._reject('app_id is not supported on the OSS backend');
    }

    const payload: JsonObject = { query };
    const filters: JsonObject = { ...(opts.filters ?? {}) };
    if (opts.userId) filters.user_id = opts.userId;
    if (opts.agentId) filters.agent_id = opts.agentId;
    if (opts.runId) filters.run_id = opts.runId;
    if (Object.keys(filters).length > 0) payload.filters = filters;
    payload.top_k = opts.topK ?? 10;
    payload.threshold = opts.threshold ?? 0.3;
    if (opts.showExpired) payload.show_expired = true;

    const result = (await this._request('POST', '/search', {
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
    return jsonObject(await this._request('GET', `/memories/${encodePathSegment(memoryId)}`));
  }

  /**
   * 按筛选与分页条件读取记忆列表。
   * @param opts - 本次操作参数。
   * @returns 当前页的记忆列表。
   */
  async listMemories(opts: ListOptions): Promise<JsonObject[]> {
    if ((opts.page ?? 1) > 1 || opts.category || opts.after || opts.before || opts.latestOnly) {
      this._reject('pagination/category/date filters are not supported on the OSS backend');
    }
    if (opts.appId) {
      this._reject('app_id is not supported on the OSS backend');
    }

    const params: Record<string, string> = {
      top_k: String(opts.pageSize ?? 100),
    };
    if (opts.userId) params.user_id = opts.userId;
    if (opts.agentId) params.agent_id = opts.agentId;
    if (opts.runId) params.run_id = opts.runId;
    if (opts.showExpired) params.show_expired = 'true';

    const result = (await this._request('GET', '/memories', {
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
    if (opts.timestamp !== undefined) {
      this._reject('--timestamp is not supported on the OSS backend');
    }
    const payload: JsonObject = {};
    if (content) payload.text = content;
    if (metadata) payload.metadata = metadata;
    if (opts.expirationDate) payload.expiration_date = opts.expirationDate;
    return jsonObject(
      await this._request('PUT', `/memories/${encodePathSegment(memoryId)}`, {
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
      if (!opts.userId && !opts.agentId && !opts.runId) {
        this._reject(
          'OSS --all requires at least one scope id (user/agent/run); project-wide reset uses POST /reset via server admin'
        );
      }
      const params: Record<string, string> = {};
      if (opts.userId) params.user_id = opts.userId;
      if (opts.agentId) params.agent_id = opts.agentId;
      if (opts.runId) params.run_id = opts.runId;
      return jsonObject(
        await this._request('DELETE', '/memories', {
          params,
        })
      );
    }
    if (!memoryId) {
      throw new Error('Either memoryId or --all is required');
    }
    if (opts.deleteLinked) {
      this._reject('--delete-linked is not supported on the OSS backend');
    }
    return jsonObject(await this._request('DELETE', `/memories/${encodePathSegment(memoryId)}`));
  }

  /**
   * 逐项删除指定实体及其关联记忆。
   * @param opts - 本次操作参数。
   * @returns 各实体删除的聚合结果。
   */
  async deleteEntities(opts: EntityIds): Promise<JsonObject> {
    if (opts.appId) {
      this._reject('app_id is not supported on the OSS backend');
    }
    const typeMap: [string, string | undefined][] = [
      ['user', opts.userId],
      ['agent', opts.agentId],
      ['run', opts.runId],
    ];
    const entities = typeMap.filter(([, v]) => v) as [string, string][];
    if (entities.length === 0) {
      throw new Error('At least one entity ID is required for deleteEntities.');
    }
    const results: JsonObject = {};
    for (const [entityType, entityId] of entities) {
      results[entityType] = jsonObject(
        await this._request(
          'DELETE',
          `/entities/${encodePathSegment(entityType)}/${encodePathSegment(entityId)}`
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
    // ponytail: 5s 探活对齐 Go OSSBackend.Ping —— status 只需快速判连通
    return jsonObject(
      await this._request('GET', '/auth/setup-status', {
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
      return { connected: true, backend: 'oss', base_url: this.baseUrl };
    } catch (e) {
      return {
        connected: false,
        backend: 'oss',
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
    if (entityType === 'apps') {
      this._reject('app entities are not supported on the OSS backend');
    }
    const result = (await this._request('GET', '/entities')) as unknown;
    const items = jsonRecords(result);
    const singular = entityType.replace(/s$/, '');
    return items.filter((e) => (e.type as string | undefined)?.toLowerCase() === singular);
  }

  /**
   * 明确拒绝 OSS 尚不支持的异步事件操作。
   */
  async listEvents(): Promise<JsonObject[]> {
    this._reject('events are not supported on the OSS backend');
  }

  /**
   * 明确拒绝 OSS 尚不支持的异步事件操作。
   * @param _eventId - 为保持 Backend 接口一致而保留的事件 ID。
   */
  async getEvent(_eventId: string): Promise<JsonObject> {
    this._reject('events are not supported on the OSS backend');
  }
}
