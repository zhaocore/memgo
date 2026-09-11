/**
 * 可选集成失败只记录操作名，不输出可能包含凭据的异常正文。
 * @param operation - 发生可选集成故障的操作名称。
 */
export function warnOptional(operation: string): void {
  process.stderr.write(
    `${JSON.stringify({ level: 'warn', event: 'optional_operation_failed', operation })}\n`
  );
}
