import { formatJsonEnvelope } from '../../output/format.js';
import { exit } from '../../runtime/exit.js';
import { isAgentMode } from '../../runtime/state.js';
/** 从本地配置读取当前代理的 default_user_id，不发网络请求。 */

import { loadConfig } from '../../config/store.js';
import { colors, printError, printInfo } from '../../output/branding.js';

/**
 * 输出当前账号及代理模式信息。
 */
export async function cmdWhoami(): Promise<void> {
  const config = loadConfig();
  const sessionId = config.platform?.defaultUserId;
  if (!sessionId) {
    printError('No default_user_id found. Run `memgo init --agent` first.');
    exit(1);
  }
  if (isAgentMode()) {
    formatJsonEnvelope({
      command: 'whoami',
      data: { default_user_id: sessionId },
    });
    return;
  }
  console.log(`Your AGENTRUSH identifier:  ${colors.brand(sessionId)}`);
  printInfo('Find your row at https://memgo.ai/agentrush');
}
