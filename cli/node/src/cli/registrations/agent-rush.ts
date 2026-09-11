import type { Command } from 'commander';
import { richFormatHelp } from '../help.js';

/**
 * 注册 agent-rush 命令，保持参数和输出合同。
 * @param program - 本次装配的 Commander 根命令。
 */
export function registerAgentRush(program: Command): void {
  const agentRush = program
    .command('agent-rush')
    .description('AGENTRUSH game commands.')
    .addHelpCommand(false)
    .configureHelp({ formatHelp: richFormatHelp });

  agentRush
    .command('add <content...>')
    .description('Submit a memory to AGENTRUSH.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo agent-rush add "I used memgo to build a coding agent"\n  $ memgo agent-rush add "Agents that remember are better agents"'
    )
    .action(async (parts: string[]) => {
      const { cmdAgentRushAdd } = await import('../commands/agent-rush.js');
      await cmdAgentRushAdd(parts.join(' '));
    });

  agentRush
    .command('search <query...>')
    .description('Search AGENTRUSH memories.')
    .addHelpText(
      'after',
      '\nExamples:\n  $ memgo agent-rush search "agents and memory and tools"\n  $ memgo agent-rush search "coding assistant"'
    )
    .action(async (parts: string[]) => {
      const { cmdAgentRushSearch } = await import('../commands/agent-rush.js');
      await cmdAgentRushSearch(parts.join(' '));
    });
}
