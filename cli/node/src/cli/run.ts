import { CommanderError } from 'commander';
import { printError } from '../output/branding.js';
import { CliExit } from '../runtime/exit.js';
import {
  createInvocationState,
  isAgentMode,
  takeNotice,
  withInvocation,
} from '../runtime/state.js';
import { createProgram } from './program.js';

/**
 * 执行一次命令；退出前清理，允许同进程再次调用。
 * @param argv - 包含 Node 与脚本路径的完整进程参数。
 * @returns 命令成功或失败对应的进程退出码。
 */
export async function runCli(argv: string[]): Promise<number> {
  return withInvocation(createInvocationState(), async () => {
    try {
      await createProgram().parseAsync(argv);
      return 0;
    } catch (error) {
      if (error instanceof CliExit || error instanceof CommanderError) return error.exitCode;
      printError(error instanceof Error ? error.message : String(error));
      return 1;
    } finally {
      const notice = takeNotice();
      if (notice && !isAgentMode()) process.stderr.write(`\n\x1b[33m🔔 ${notice}\x1b[0m\n\n`);
    }
  });
}
