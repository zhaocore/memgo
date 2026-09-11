/** 已输出诊断的命令通过此信号结束，入口负责退出码。 */
export class CliExit extends Error {
  /**
   * 携带退出码结束命令，由入口统一处理进程状态。
   * @param exitCode - 由 CLI 入口处理的退出码。
   */
  constructor(readonly exitCode: number) {
    super(`CLI exited with code ${exitCode}`);
    this.name = 'CliExit';
  }
}
/**
 * 终止当前命令，不直接结束进程，以便 finally 清理资源。
 * @param code - 交由入口处理的进程退出码。
 */
export function exit(code: number): never {
  throw new CliExit(code);
}
