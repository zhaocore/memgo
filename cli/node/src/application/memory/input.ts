import { type JsonObject, jsonObject } from '../../backend/json.js';
/**
 * 消息输入必须是对象数组，具备 role 和 content，额外协议字段保留。
 * @param text - 需要解析的 JSON 文本。
 * @returns 通过角色与内容校验的消息列表。
 */
export function parseMessages(text: string): JsonObject[] {
  const data: unknown = JSON.parse(text);
  if (!Array.isArray(data)) throw new Error('Messages must be a JSON array');
  return data.map((value) => {
    const message = jsonObject(value);
    if (
      typeof message.role !== 'string' ||
      !(typeof message.content === 'string' || Array.isArray(message.content))
    )
      throw new Error('Each message requires role and content');
    return message;
  });
}
/**
 * 开放 JSON 对象仍须拒绝 null、数组及标量，不能仅靠类型断言。
 * @param text - 需要解析的 JSON 文本。
 * @returns 通过校验的 JSON 对象。
 */
export function parseObject(text: string): JsonObject {
  return jsonObject(JSON.parse(text));
}
/**
 * 自定义分类是字符串值的对象数组，保持既有请求格式。
 * @param text - 需要解析的 JSON 文本。
 * @returns 仅包含字符串描述的分类列表。
 */
export function parseCategories(text: string): Record<string, string>[] {
  const data: unknown = JSON.parse(text);
  if (!Array.isArray(data)) throw new Error('Custom categories must be an array');
  return data.map((value) => {
    const object = jsonObject(value);
    if (!Object.values(object).every((item) => typeof item === 'string'))
      throw new Error('Category descriptions must be strings');
    return object as Record<string, string>;
  });
}
