import type { Command } from 'commander';

/**
 * 注册 whoami 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerWhoami(program: Command): void {
  program
    .command('whoami')
    .description("Print the active agent's AGENTRUSH identifier.")
    .action(async () => {
      const { cmdWhoami } = await import('../commands/whoami.js');
      await cmdWhoami();
    });
}
