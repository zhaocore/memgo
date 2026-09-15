/**
 * MemGo DeepSeek 插件本地遥测。
 *
 * 事件以 JSONL 逐行追加到 ~/.memgo/deepseek-plugin-telemetry.jsonl, 不对外
 * 发送: 原 PostHog 发送(POSTHOG_* key、batch URL、fetch 上传)与账号邮箱
 * 解析(/v1/ping/)已删除。
 *
 * 事件只携带工具名、耗时、计数与粗粒度失败类型; 绝不包含查询、记忆文本、
 * 过滤条件或 API key。MEMGO_TELEMETRY=false 关闭。
 */
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const PLUGIN_VERSION = ((): string => {
  try {
    return JSON.parse(
      fs.readFileSync(new URL("../package.json", import.meta.url), "utf-8"),
    ).version;
  } catch {
    return "unknown";
  }
})();

// 托管平台用它给后端遥测归因(KNOWN_EVENT_SOURCES allowlist)。MemGo 本地化
// 后 OSS 面无 source 字段, 该常量仅作为本地 spool 的标注, 不传给 server。
const SOURCE = "MEMGO_HARNESS";

/** 本地 spool 文件名(位于 ~/.memgo/ 下)。 */
const SPOOL_FILE = "deepseek-plugin-telemetry.jsonl";

export function isTelemetryEnabled(): boolean {
  const value = process.env.MEMGO_TELEMETRY?.toLowerCase();
  return value !== "false" && value !== "0" && value !== "no" && value !== "off";
}

/**
 * 把任意错误归约为粗粒度类型。客户端抛出的 ApiError 自带 kind 时直接读取;
 * 其余(第三方错误)按消息文本兜底分类。
 */
export function errorKind(error: unknown): string {
  const kind = (error as { kind?: unknown } | null)?.kind;
  if (typeof kind === "string" && kind) return kind;
  const text = (error instanceof Error ? error.message : String(error)).toLowerCase();
  if (text.includes("timeout") || text.includes("aborted")) return "timeout";
  if (text.includes("401") || text.includes("403") || text.includes("unauthor")) return "auth";
  if (text.includes("429") || text.includes("rate limit")) return "rate-limited";
  if (/50[0234]/.test(text)) return "server-error";
  if (text.includes("400") || text.includes("422")) return "bad-request";
  if (text.includes("fetch failed") || text.includes("enotfound")) return "network";
  return error instanceof Error ? error.constructor.name : "other";
}

/**
 * 追加一条事件到本地 spool。绝不抛出, 绝不阻断工具调用。
 *
 * 原托管 SDK 时代第三参为 telemetryId 载体(账号邮箱标识与匿名归并);
 * 本地化后不再需要, 保留该参数仅为兼容调用点(apply 仍传入 client)。
 */
export function captureEvent(
  event: string,
  properties: Record<string, unknown>,
  _client?: unknown,
): void {
  if (!isTelemetryEnabled()) return;
  try {
    const line = JSON.stringify({
      event,
      timestamp: new Date().toISOString(),
      properties: {
        source: SOURCE,
        language: "node",
        plugin_version: PLUGIN_VERSION,
        node_version: process.version,
        os: process.platform,
        ...properties,
      },
    });
    const target = spoolPath();
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.appendFileSync(target, line + "\n", "utf-8");
  } catch {
    /* 遥测绝不把自己的失败暴露出去 */
  }
}

function spoolPath(): string {
  return path.join(os.homedir(), ".memgo", SPOOL_FILE);
}
