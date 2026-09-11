import type { JsonObject } from './json.js';
export interface AddOptions {
  userId?: string;
  agentId?: string;
  appId?: string;
  runId?: string;
  metadata?: JsonObject;
  immutable?: boolean;
  infer?: boolean;
  expires?: string;
  customInstructions?: string;
  agentCustomInstructions?: string;
  customCategories?: Record<string, string>[];
  structuredDataSchema?: JsonObject;
  timestamp?: number;
}

export interface SearchOptions {
  userId?: string;
  agentId?: string;
  appId?: string;
  runId?: string;
  topK?: number;
  threshold?: number;
  rerank?: boolean;
  keyword?: boolean;
  filters?: JsonObject;
  fields?: string[];
  showExpired?: boolean;
  referenceDate?: string | number;
  latestOnly?: boolean;
}

export interface ListOptions {
  userId?: string;
  agentId?: string;
  appId?: string;
  runId?: string;
  page?: number;
  pageSize?: number;
  category?: string;
  after?: string;
  before?: string;
  showExpired?: boolean;
  latestOnly?: boolean;
}

export interface DeleteOptions {
  all?: boolean;
  userId?: string;
  agentId?: string;
  appId?: string;
  runId?: string;
  deleteLinked?: boolean;
}

export interface UpdateOptions {
  expirationDate?: string;
  timestamp?: number;
}

export interface EntityIds {
  userId?: string;
  agentId?: string;
  appId?: string;
  runId?: string;
}

export interface Backend {
  add(
    content: string | undefined,
    messages: JsonObject[] | undefined,
    opts: AddOptions
  ): Promise<JsonObject | JsonObject[]>;

  search(query: string, opts: SearchOptions): Promise<JsonObject[]>;

  get(memoryId: string): Promise<JsonObject>;

  listMemories(opts: ListOptions): Promise<JsonObject[]>;

  update(
    memoryId: string,
    content: string | undefined,
    metadata: JsonObject | undefined,
    opts: UpdateOptions
  ): Promise<JsonObject>;

  delete(memoryId: string | undefined, opts: DeleteOptions): Promise<JsonObject>;

  deleteEntities(opts: EntityIds): Promise<JsonObject>;

  ping(): Promise<JsonObject>;

  status(opts: { userId?: string; agentId?: string }): Promise<JsonObject>;

  entities(entityType: string): Promise<JsonObject[]>;

  listEvents(): Promise<JsonObject[]>;

  getEvent(eventId: string): Promise<JsonObject>;
}
