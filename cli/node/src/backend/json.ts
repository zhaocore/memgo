/** 外部 JSON 的开放字段，仅允许可序列化值。 */
export type JsonValue = string | number | boolean | null | JsonObject | JsonValue[];
export interface JsonObject {
  [key: string]: JsonValue | undefined;
}

/**
 * 校验整个 JSON 树，拒绝非 JSON 值；返回值与输入不共享可变对象。
 * @param value - 待处理的字段值。
 * @returns 递归校验后的 JSON 值。
 */
export function parseJsonValue(value: unknown): JsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return value;
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (Array.isArray(value)) return value.map(parseJsonValue);
  if (typeof value === 'object' && value !== null) {
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, parseJsonValue(item)])
    );
  }
  throw new Error('Response contains a value that is not valid JSON');
}
/**
 * 对象型协议边界不接受数组、null 或标量。
 * @param value - 待处理的字段值。
 * @returns 通过对象形状校验的 JSON 数据。
 */
export function jsonObject(value: unknown): JsonObject {
  const parsed = parseJsonValue(value);
  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed))
    throw new Error('Expected a JSON object');
  return parsed;
}
/**
 * 列表接口允许裸数组及已存在的 results/memories 信封，缺少集合时明确失败。
 * @param value - 待处理的字段值。
 * @returns 从数组或列表信封中提取的对象列表。
 */
export function jsonRecords(value: unknown): JsonObject[] {
  const parsed = parseJsonValue(value);
  const records = Array.isArray(parsed)
    ? parsed
    : parsed && typeof parsed === 'object'
      ? (parsed.results ?? parsed.memories)
      : undefined;
  if (!Array.isArray(records))
    throw new Error('Expected an array or a results/memories array envelope');
  return records.map(jsonObject);
}
/**
 * 添加接口兼容对象和对象数组两种既有成功形状。
 * @param value - 待处理的字段值。
 * @returns 保留单对象或对象数组形状的结果。
 */
export function jsonResult(value: unknown): JsonObject | JsonObject[] {
  return Array.isArray(value) ? value.map(jsonObject) : jsonObject(value);
}
