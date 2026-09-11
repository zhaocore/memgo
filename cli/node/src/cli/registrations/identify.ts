import type { Command } from 'commander';

/**
 * 注册 identify 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerIdentify(program: Command): void {
  program
    .command('identify <name>')
    .description(
      "Tag your active Agent Mode key with the AI agent that's using it (e.g. claude-code, cursor)."
    )
    .action(async (name: string) => {
      const { runIdentify } = await import('../commands/identify.js');
      await runIdentify(name);
    });
}
