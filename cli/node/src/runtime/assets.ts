import { fileURLToPath } from 'node:url';
/**
 * 源码与构建目录层级不同，构建常量固定资源路径，不进行文件探测回退。
 * @returns 适配源码或构建布局的发送器绝对路径。
 */
export function telemetrySenderPath(): string {
  const relative =
    typeof __CLI_BUILD__ !== 'undefined' && __CLI_BUILD__
      ? '../telemetry-sender.cjs'
      : '../../telemetry-sender.cjs';
  return fileURLToPath(new URL(relative, import.meta.url));
}
