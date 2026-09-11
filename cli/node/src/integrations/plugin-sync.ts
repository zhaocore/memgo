import type { JsonObject } from '../backend/json.js';
import { warnOptional } from '../runtime/warning.js';
/** 同步已有 Claude 环境字段及 shell 导出。保留其他字段，以临时文件加重命名写入；相同密钥重复同步无变化。不创建新配置项。 */

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const CLAUDE_SETTINGS = path.join(os.homedir(), '.claude', 'settings.json');
const SHELL_RCS = [
  path.join(os.homedir(), '.zshrc'),
  path.join(os.homedir(), '.bashrc'),
  path.join(os.homedir(), '.bash_profile'),
];

// 使用空格和制表符匹配，保留文件末尾换行。

const RC_LINE_RE =
  /^([ \t]*export[ \t]+MEMGO_API_KEY[ \t]*=[ \t]*)(["']?)([^"'\n]*)(["']?)[ \t]*$/m;

/**
 * 将密钥同步到已有插件及 shell 配置条目。
 * @param apiKey - 访问服务的 API 密钥。
 * @returns 实际更新的配置文件路径。
 */
export function syncApiKey(apiKey: string): string[] {
  if (!apiKey) return [];
  const updated: string[] = [];
  if (updateClaudeSettings(CLAUDE_SETTINGS, apiKey)) {
    updated.push(CLAUDE_SETTINGS);
  }
  for (const rc of SHELL_RCS) {
    if (updateShellRc(rc, apiKey)) updated.push(rc);
  }
  return updated;
}

/**
 * 内部测试入口，业务调用应使用 syncApiKey。
 * @param filePath - 目标文件路径。
 * @param apiKey - 访问服务的 API 密钥。
 * @returns 是否更新了已有密钥条目。
 */
export function updateClaudeSettings(filePath: string, apiKey: string): boolean {
  if (!fs.existsSync(filePath)) return false;
  let raw: string;
  let data: JsonObject;
  try {
    raw = fs.readFileSync(filePath, 'utf-8');
    data = JSON.parse(raw);
  } catch {
    warnOptional('plugin_sync_read');
    return false;
  }
  const env = data.env;
  if (!env || typeof env !== 'object' || !('MEMGO_API_KEY' in env)) {
    return false; // 没有已有条目，不创建。
  }
  const envObj = env as Record<string, string>;
  if (envObj.MEMGO_API_KEY === apiKey) return false; // 内容已经同步。
  envObj.MEMGO_API_KEY = apiKey;
  atomicWriteText(filePath, `${JSON.stringify(data, null, 2)}\n`);
  return true;
}

/**
 * 内部测试入口，业务调用应使用 syncApiKey。
 * @param filePath - 目标文件路径。
 * @param apiKey - 访问服务的 API 密钥。
 * @returns 是否更新了已有密钥导出行。
 */
export function updateShellRc(filePath: string, apiKey: string): boolean {
  if (!fs.existsSync(filePath)) return false;
  let text: string;
  try {
    text = fs.readFileSync(filePath, 'utf-8');
  } catch {
    warnOptional('plugin_sync_read');
    return false;
  }
  const match = text.match(RC_LINE_RE);
  if (!match) return false; // 没有已有导出行。
  if (match[3] === apiKey) return false;
  const newText = text.replace(RC_LINE_RE, (_full, prefix) => `${prefix}"${apiKey}"`);
  atomicWriteText(filePath, newText);
  return true;
}

/**
 * 通过同目录临时文件替换目标文件，并保留原文件权限。
 * @param filePath - 目标文件路径。
 * @param content - 待写入的内容。
 */
function atomicWriteText(filePath: string, content: string): void {
  const dir = path.dirname(filePath);
  const tmp = path.join(dir, `.${path.basename(filePath)}.${process.pid}.tmp`);
  try {
    fs.writeFileSync(tmp, content, 'utf-8');
    // 保留原文件权限。
    if (fs.existsSync(filePath)) {
      try {
        const mode = fs.statSync(filePath).mode & 0o777;
        fs.chmodSync(tmp, mode);
      } catch {
        warnOptional('plugin-sync');
      }
    }
    fs.renameSync(tmp, filePath);
  } catch (err) {
    try {
      fs.unlinkSync(tmp);
    } catch {
      warnOptional('plugin-sync');
    }
    throw err;
  }
}
