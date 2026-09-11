import { createRequire } from 'node:module';

// tsup 在构建时替换 __CLI_VERSION__，配置见 tsup.config.ts。
// 开发模式通过 package.json 读取版本。
// typeof 可安全检查未声明的构建变量。
export const CLI_VERSION: string =
  typeof __CLI_VERSION__ !== 'undefined'
    ? (__CLI_VERSION__ as string)
    : (createRequire(import.meta.url)('../package.json') as { version: string }).version;
