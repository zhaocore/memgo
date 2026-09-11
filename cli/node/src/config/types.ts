/** 配置内存模型；持久化字段保持下划线命名。 */
export interface PlatformConfig {
  apiKey: string;
  baseUrl: string;
  userEmail: string;
  // 尚未认领的 Agent Mode 账号。
  agentMode: boolean; // 密钥属于未认领账号时为真。
  createdVia: string; // 创建方式：agent_mode、email、api_key 或 existing_key。
  agentCaller: string; // 通过 --agent-caller 声明的标准代理名称。
  claimedAt: string; // 认领完成的 ISO 时间。
  defaultUserId: string; // 初始化返回的 user_<slug>，用作默认范围。
}

export interface DefaultsConfig {
  userId: string;
  agentId: string;
  appId: string;
  runId: string;
}

export interface TelemetryConfig {
  anonymousId: string;
}

export interface AgentRushConfig {
  // 用户确认公开记忆提示的 ISO 时间。
  // 首次交互确认前为空。
  acknowledgedAt: string;
}

export interface MemGoConfig {
  version: number;
  defaults: DefaultsConfig;
  platform: PlatformConfig;
  telemetry: TelemetryConfig;
  agentRush: AgentRushConfig;
}
