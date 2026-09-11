/** 检测代理环境，仅触发自动初始化；身份只能由 --agent-caller 显式声明，不能从环境猜测。 */

const AGENT_CALLER_ENV: ReadonlyArray<readonly [string, readonly string[]]> = [
  ['claude-code', ['CLAUDECODE', 'CLAUDE_CODE']],
  ['cursor', ['CURSOR_AGENT', 'CURSOR_SESSION_ID']],
  ['codex', ['CODEX_CLI', 'OPENAI_CODEX']],
  ['cline', ['CLINE_AGENT', 'CLINE']],
  ['continue', ['CONTINUE_AGENT', 'CONTINUE_SESSION']],
  ['aider', ['AIDER_SESSION']],
  ['goose', ['GOOSE_AGENT']],
  ['windsurf', ['WINDSURF_AGENT']],
] as const;

/**
 * 根据已知环境变量识别代理调用方。
 * @returns 识别到的代理名称；未知环境返回空值。
 */
export function detectAgentCaller(): string | null {
  for (const [name, envVars] of AGENT_CALLER_ENV) {
    if (envVars.some((v) => process.env[v])) {
      return name;
    }
  }
  return null;
}
