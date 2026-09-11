import { AsyncLocalStorage } from 'node:async_hooks';
import fs from 'node:fs';

/** 单次调用的终端状态；仅运行时边界允许更新。 */
export interface InvocationState {
  agentMode: boolean;
  currentCommand: string;
  pendingNotice: string;
}
const invocations = new AsyncLocalStorage<InvocationState>();

/**
 * 创建独立上下文，避免多次执行或并发命令相互污染。
 * @returns 独立的调用状态对象。
 */
export function createInvocationState(): InvocationState {
  return { agentMode: false, currentCommand: '', pendingNotice: '' };
}
/**
 * 在调用范围内装配终端副作用依赖。
 * @param state - 本次调用独立持有的状态。
 * @param action - 需要在指定上下文内执行的函数。
 * @returns 在指定上下文中执行的函数返回值。
 */
export function withInvocation<T>(state: InvocationState, action: () => T): T {
  return invocations.run(state, action);
}
/**
 * 为测试或嵌入式调用绑定当前执行上下文。
 * @param state - 本次调用独立持有的状态。
 */
export function enterInvocation(state: InvocationState): void {
  invocations.enterWith(state);
}
/**
 * 读取本次调用状态；缺少上下文时明确报错。
 * @returns 当前异步调用上下文中的状态。
 */
function current(): InvocationState {
  const state = invocations.getStore();
  if (!state) throw new Error('CLI invocation context is missing. Use withInvocation().');
  return state;
}
/**
 * 读取本次调用的代理输出开关。
 * @returns 当前是否使用代理输出模式。
 */
export function isAgentMode(): boolean {
  return current().agentMode;
}
/**
 * 设置本次调用的代理输出开关。
 * @param value - 是否采用代理输出模式。
 */
export function setAgentMode(value: boolean): void {
  current().agentMode = value;
}
/**
 * 读取当前命令名称。
 * @returns 当前命令名称。
 */
export function getCurrentCommand(): string {
  return current().currentCommand;
}
/**
 * 记录当前命令名称供输出信封使用。
 * @param name - 调用方或命令名称。
 */
export function setCurrentCommand(name: string): void {
  current().currentCommand = name;
}
/**
 * 后端提示每条命令只输出一次，不随请求数量重复。
 * @param notice - 等待输出的平台通知。
 */
export function captureNotice(notice: string | null | undefined): void {
  if (notice) current().pendingNotice = notice;
}
/**
 * 取出待展示通知并清空本次调用中的缓存。
 * @returns 本次取出的通知，未设置时为空字符串。
 */
export function takeNotice(): string {
  const state = current();
  const notice = state.pendingNotice;
  state.pendingNotice = '';
  return notice;
}
/**
 * 只读实际管道或重定向文件，避免 /dev/null 和 socket 的 EAGAIN。
 * @returns 当前是否允许并检测到文件或管道输入。
 */
export function stdinIsPiped(): boolean {
  if (isAgentMode()) return false;
  const stat = fs.fstatSync(0);
  return stat.isFIFO() || stat.isFile();
}
