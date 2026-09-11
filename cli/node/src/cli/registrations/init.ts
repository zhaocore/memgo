import type { Command } from 'commander';
import { setAgentMode } from '../../runtime/state.js';

/**
 * 注册 init 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerInit(program: Command): void {
  program
    .command('init')
    .description('Interactive setup wizard for memgo CLI.')
    .option('--api-key <key>', 'API key (skip prompt).')
    .option('-u, --user-id <id>', 'Default user ID (skip prompt).')
    .option('--email <email>', 'Login via email verification code.')
    .option('--code <code>', 'Verification code (use with --email for non-interactive login).')
    .option('--force', 'Overwrite existing config without confirmation.', false)
    .option('--agent', 'Bootstrap an unattended Agent Mode account (no email required).', false)
    .option('--source <channel>', 'Channel attribution for signup (e.g. github, hn, ph).')
    .option(
      '--agent-caller <name>',
      'Self-declared agent identity (e.g. claude-code, cursor). Used with --agent to attribute Agent Mode signups.'
    )
    // init --agent --json 与根命令 JSON 模式一致。

    .option('--json', 'Output as JSON (alias for global `--json`).', false)
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo init\n  $ memgo init --api-key m0-xxx --user-id alice\n  $ memgo init --email you@example.com\n  $ memgo init --email you@example.com --code 123456\n  $ memgo init --agent             # Bootstrap an Agent Mode account (unattended)\n  $ memgo init --email you@example.com  # Claims an existing Agent Mode key when one is present'
    )
    .action(async (opts) => {
      // init 的 --json 与全局输出选项保持一致。

      if (opts.json) setAgentMode(true);
      const { runInit } = await import('../../application/onboarding/init.js');
      await runInit({
        apiKey: opts.apiKey,
        userId: opts.userId,
        email: opts.email,
        code: opts.code,
        force: opts.force,
        agent: opts.agent,
        source: opts.source,
        agentCaller: opts.agentCaller,
      });
    });
}
