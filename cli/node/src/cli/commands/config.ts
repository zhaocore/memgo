import { exit } from '../../runtime/exit.js';
/** 配置读取、展示与更新命令。 */

import Table from 'cli-table3';
import {
  getNestedValue,
  loadConfig,
  redactKey,
  saveConfig,
  setNestedValue,
} from '../../config/store.js';
import { colors, printError, printSuccess } from '../../output/branding.js';
import { formatAgentEnvelope } from '../../output/format.js';
import { isAgentMode, setCurrentCommand } from '../../runtime/state.js';

const { brand, accent, dim } = colors;

/**
 * 展示当前配置，对敏感密钥进行脱敏。
 * @param opts - 本次操作选项。
 * @param opts.output - 终端输出格式。
 */
export function cmdConfigShow(opts: { output?: string }): void {
  setCurrentCommand('config show');
  const config = loadConfig();

  if (opts.output === 'agent' || opts.output === 'json') {
    formatAgentEnvelope({
      command: 'config show',
      data: {
        defaults: {
          user_id: config.defaults.userId || null,
          agent_id: config.defaults.agentId || null,
          app_id: config.defaults.appId || null,
          run_id: config.defaults.runId || null,
        },
        platform: {
          api_key: redactKey(config.platform.apiKey),
          base_url: config.platform.baseUrl,
        },
      },
    });
    return;
  }

  console.log();
  console.log(`  ${brand('◆ memgo Configuration')}\n`);

  const table = new Table({
    head: [accent('Key'), accent('Value')],
    style: { head: [], border: [] },
  });

  // 默认实体配置。
  table.push(['defaults.user_id', config.defaults.userId || dim('(not set)')]);
  table.push(['defaults.agent_id', config.defaults.agentId || dim('(not set)')]);
  table.push(['defaults.app_id', config.defaults.appId || dim('(not set)')]);
  table.push(['defaults.run_id', config.defaults.runId || dim('(not set)')]);
  table.push(['', '']);

  // 平台配置。
  table.push(['platform.api_key', redactKey(config.platform.apiKey)]);
  table.push(['platform.base_url', config.platform.baseUrl]);

  console.log(table.toString());
  console.log();
}

/**
 * 读取指定配置字段，未知字段以失败状态退出。
 * @param key - 需要读取或更新的配置字段名。
 */
export function cmdConfigGet(key: string): void {
  setCurrentCommand('config get');
  const config = loadConfig();
  const value = getNestedValue(config, key);

  if (value === undefined) {
    printError(`Unknown config key: ${key}`);
    exit(1);
  } else {
    // 敏感字段只输出脱敏值。
    const displayValue =
      key.includes('api_key') || key.split('.').pop() === 'key'
        ? redactKey(String(value))
        : String(value);
    if (isAgentMode()) {
      formatAgentEnvelope({
        command: 'config get',
        data: { key, value: displayValue },
      });
    } else {
      console.log(displayValue);
    }
  }
}

/**
 * 校验并保存指定配置字段。
 * @param key - 需要读取或更新的配置字段名。
 * @param value - 待处理的字段值。
 */
export function cmdConfigSet(key: string, value: string): void {
  setCurrentCommand('config set');
  const config = loadConfig();
  if (getNestedValue(config, key) !== undefined) {
    saveConfig(setNestedValue(config, key, value));
    const display = key.includes('key') ? redactKey(value) : value;
    if (isAgentMode()) {
      formatAgentEnvelope({
        command: 'config set',
        data: { key, value: display },
      });
    } else {
      printSuccess(`${key} = ${display}`);
    }
  } else {
    printError(`Unknown config key: ${key}`);
    exit(1);
  }
}
