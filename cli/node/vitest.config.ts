import { createRequire } from 'node:module';
import { defineConfig } from 'vitest/config';

const _require = createRequire(import.meta.url);
const pkg = _require('./package.json') as { version: string };

export default defineConfig({
  define: {
    __CLI_VERSION__: JSON.stringify(pkg.version),
  },
  test: {
    setupFiles: ['tests/runtime-setup.ts'],
    // 集成测试执行构建产物，预留网络超时和子进程启动时间。

    testTimeout: 30_000,
  },
});
