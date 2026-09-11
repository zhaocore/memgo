import { printError } from '../../output/branding.js';
import { exit } from '../../runtime/exit.js';

/**
 * 校验到期日格式及未来日期约束，失败时结束命令。
 * @param value - 到期日期，格式为 YYYY-MM-DD。
 */
export function _validateExpires(value: string): void {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    printError('Invalid date format for --expires. Use YYYY-MM-DD (e.g. 2025-12-31).');
    exit(1);
  }
  if (new Date(value) <= new Date()) {
    printError('--expires date must be in the future.');
    exit(1);
  }
}
