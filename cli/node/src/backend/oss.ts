/**
 * OSS (self-hosted MemGo server) backend — doc-02 contract, X-API-Key auth.
 *
 * Pure addition: no command-layer changes; the factory in base.ts picks this
 * backend whenever base_url is NOT the platform host (doc-01 §4.3).
 * Behavioral parity with Go CLI backend_oss.go: unsupported platform-only
 * options fail loudly instead of silently degrading.
 */

import type { PlatformConfig } from "../config.js";
import {
	APIError,
	type AddOptions,
	AuthError,
	type Backend,
	type DeleteOptions,
	type EntityIds,
	type ListOptions,
	NotFoundError,
	type SearchOptions,
	type UpdateOptions,
} from "./base.js";

function encodePathSegment(value: unknown): string {
	return encodeURIComponent(String(value));
}

export class OSSBackend implements Backend {
	private baseUrl: string;
	private headers: Record<string, string>;

	constructor(config: PlatformConfig) {
		this.baseUrl = config.baseUrl.replace(/\/+$/, "");
		this.headers = {
			"X-API-Key": config.apiKey,
			"Content-Type": "application/json",
		};
	}

	// ponytail: 600s 超时对齐 Go OSSBackend —— infer=true 走真实 LLM 时远超普通 API 延迟
	private async _request(
		method: string,
		path: string,
		opts?: {
			json?: unknown;
			params?: Record<string, string>;
			timeoutMs?: number;
		},
	): Promise<unknown> {
		let url = `${this.baseUrl}${path}`;
		if (opts?.params) {
			const qs = new URLSearchParams(opts.params).toString();
			url += `?${qs}`;
		}

		const fetchOpts: RequestInit = {
			method,
			headers: this.headers,
			signal: AbortSignal.timeout(opts?.timeoutMs ?? 600_000),
		};
		if (opts?.json) {
			fetchOpts.body = JSON.stringify(opts.json);
		}

		const resp = await fetch(url, fetchOpts);

		if (resp.status === 401) {
			throw new AuthError();
		}
		if (resp.status === 403) {
			throw new AuthError(`Forbidden: ${await this._detail(resp)}`);
		}
		if (resp.status === 404) {
			throw new NotFoundError(path);
		}
		if (!resp.ok) {
			throw new APIError(
				path,
				`HTTP ${resp.status}: ${await this._detail(resp)}`,
			);
		}
		if (resp.status === 204) {
			return {};
		}
		try {
			return await resp.json();
		} catch {
			return {};
		}
	}

	private async _detail(resp: Response): Promise<string> {
		try {
			const body = (await resp.json()) as Record<string, unknown>;
			return ((body.detail ?? body.message) as string) ?? resp.statusText;
		} catch {
			return resp.statusText;
		}
	}

	private _reject(msg: string): never {
		throw new APIError("oss-backend", msg);
	}

	async add(
		content?: string,
		messages?: Record<string, unknown>[],
		opts: AddOptions = {},
	): Promise<Record<string, unknown>> {
		if (opts.appId) {
			this._reject("app_id is not supported on the OSS backend");
		}
		if (opts.immutable) {
			this._reject("--immutable is not supported on the OSS backend");
		}
		if (
			opts.structuredDataSchema ||
			opts.customCategories ||
			opts.agentCustomInstructions
		) {
			this._reject(
				"platform-only extraction options are not supported on the OSS backend",
			);
		}

		const payload: Record<string, unknown> = {};
		if (messages) {
			payload.messages = messages;
		} else if (content) {
			payload.messages = [{ role: "user", content }];
		}
		if (opts.userId) payload.user_id = opts.userId;
		if (opts.agentId) payload.agent_id = opts.agentId;
		if (opts.runId) payload.run_id = opts.runId;
		if (opts.metadata) payload.metadata = opts.metadata;
		if (opts.infer === false) payload.infer = false;
		if (opts.expires) payload.expiration_date = opts.expires;
		if (opts.customInstructions) payload.prompt = opts.customInstructions;

		return (await this._request("POST", "/memories", {
			json: payload,
		})) as Record<string, unknown>;
	}

	async search(
		query: string,
		opts: SearchOptions = {},
	): Promise<Record<string, unknown>[]> {
		if (opts.keyword) {
			this._reject("--keyword is not supported on the OSS backend");
		}
		if (opts.referenceDate !== undefined || opts.latestOnly) {
			this._reject(
				"--reference-date/--latest-only are not supported on the OSS backend",
			);
		}
		if (opts.appId) {
			this._reject("app_id is not supported on the OSS backend");
		}

		const payload: Record<string, unknown> = { query };
		const filters: Record<string, unknown> = { ...(opts.filters ?? {}) };
		if (opts.userId) filters.user_id = opts.userId;
		if (opts.agentId) filters.agent_id = opts.agentId;
		if (opts.runId) filters.run_id = opts.runId;
		if (Object.keys(filters).length > 0) payload.filters = filters;
		payload.top_k = opts.topK ?? 10;
		payload.threshold = opts.threshold ?? 0.3;
		if (opts.showExpired) payload.show_expired = true;

		const result = (await this._request("POST", "/search", {
			json: payload,
		})) as unknown;
		if (Array.isArray(result)) return result;
		return ((result as Record<string, unknown>).results ?? []) as Record<
			string,
			unknown
		>[];
	}

	async get(memoryId: string): Promise<Record<string, unknown>> {
		return (await this._request(
			"GET",
			`/memories/${encodePathSegment(memoryId)}`,
		)) as Record<string, unknown>;
	}

	async listMemories(
		opts: ListOptions = {},
	): Promise<Record<string, unknown>[]> {
		if (
			(opts.page ?? 1) > 1 ||
			opts.category ||
			opts.after ||
			opts.before ||
			opts.latestOnly
		) {
			this._reject(
				"pagination/category/date filters are not supported on the OSS backend",
			);
		}
		if (opts.appId) {
			this._reject("app_id is not supported on the OSS backend");
		}

		const params: Record<string, string> = {
			top_k: String(opts.pageSize ?? 100),
		};
		if (opts.userId) params.user_id = opts.userId;
		if (opts.agentId) params.agent_id = opts.agentId;
		if (opts.runId) params.run_id = opts.runId;
		if (opts.showExpired) params.show_expired = "true";

		const result = (await this._request("GET", "/memories", {
			params,
		})) as unknown;
		if (Array.isArray(result)) return result;
		return ((result as Record<string, unknown>).results ?? []) as Record<
			string,
			unknown
		>[];
	}

	async update(
		memoryId: string,
		content?: string,
		metadata?: Record<string, unknown>,
		opts: UpdateOptions = {},
	): Promise<Record<string, unknown>> {
		if (opts.timestamp !== undefined) {
			this._reject("--timestamp is not supported on the OSS backend");
		}
		const payload: Record<string, unknown> = {};
		if (content) payload.text = content;
		if (metadata) payload.metadata = metadata;
		if (opts.expirationDate) payload.expiration_date = opts.expirationDate;
		return (await this._request(
			"PUT",
			`/memories/${encodePathSegment(memoryId)}`,
			{ json: payload },
		)) as Record<string, unknown>;
	}

	async delete(
		memoryId?: string,
		opts: DeleteOptions = {},
	): Promise<Record<string, unknown>> {
		if (opts.all) {
			if (!opts.userId && !opts.agentId && !opts.runId) {
				this._reject(
					"OSS --all requires at least one scope id (user/agent/run); project-wide reset uses POST /reset via server admin",
				);
			}
			const params: Record<string, string> = {};
			if (opts.userId) params.user_id = opts.userId;
			if (opts.agentId) params.agent_id = opts.agentId;
			if (opts.runId) params.run_id = opts.runId;
			return (await this._request("DELETE", "/memories", {
				params,
			})) as Record<string, unknown>;
		}
		if (!memoryId) {
			throw new Error("Either memoryId or --all is required");
		}
		if (opts.deleteLinked) {
			this._reject("--delete-linked is not supported on the OSS backend");
		}
		return (await this._request(
			"DELETE",
			`/memories/${encodePathSegment(memoryId)}`,
		)) as Record<string, unknown>;
	}

	async deleteEntities(opts: EntityIds): Promise<Record<string, unknown>> {
		if (opts.appId) {
			this._reject("app_id is not supported on the OSS backend");
		}
		const typeMap: [string, string | undefined][] = [
			["user", opts.userId],
			["agent", opts.agentId],
			["run", opts.runId],
		];
		const entities = typeMap.filter(([, v]) => v) as [string, string][];
		if (entities.length === 0) {
			throw new Error("At least one entity ID is required for deleteEntities.");
		}
		const results: Record<string, unknown> = {};
		for (const [entityType, entityId] of entities) {
			results[entityType] = (await this._request(
				"DELETE",
				`/entities/${encodePathSegment(entityType)}/${encodePathSegment(entityId)}`,
			)) as Record<string, unknown>;
		}
		return results;
	}

	async ping(): Promise<Record<string, unknown>> {
		// ponytail: 5s 探活对齐 Go OSSBackend.Ping —— status 只需快速判连通
		return (await this._request("GET", "/auth/setup-status", {
			timeoutMs: 5_000,
		})) as Record<string, unknown>;
	}

	async status(
		_opts: { userId?: string; agentId?: string } = {},
	): Promise<Record<string, unknown>> {
		try {
			await this.ping();
			return { connected: true, backend: "oss", base_url: this.baseUrl };
		} catch (e) {
			return {
				connected: false,
				backend: "oss",
				error: e instanceof Error ? e.message : String(e),
			};
		}
	}

	async entities(entityType: string): Promise<Record<string, unknown>[]> {
		if (entityType === "apps") {
			this._reject("app entities are not supported on the OSS backend");
		}
		const result = (await this._request("GET", "/entities")) as unknown;
		let items: Record<string, unknown>[];
		if (Array.isArray(result)) {
			items = result;
		} else {
			items = ((result as Record<string, unknown>).results ?? []) as Record<
				string,
				unknown
			>[];
		}
		const singular = entityType.replace(/s$/, "");
		return items.filter(
			(e) => (e.type as string | undefined)?.toLowerCase() === singular,
		);
	}

	async listEvents(): Promise<Record<string, unknown>[]> {
		this._reject("events are not supported on the OSS backend");
	}

	async getEvent(_eventId: string): Promise<Record<string, unknown>> {
		this._reject("events are not supported on the OSS backend");
	}
}
