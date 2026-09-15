import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  dts: true,
  sourcemap: true,
  clean: true,
  // harness 运行时(由 host 提供)与 Node 内置模块不进 bundle; 插件无其他外部运行时依赖。
  external: [/^node:/, /^@deepseek-ai\//],
});
