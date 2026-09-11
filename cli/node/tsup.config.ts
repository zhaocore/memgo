import { createRequire } from 'node:module';
import { defineConfig } from 'tsup';

const _require = createRequire(import.meta.url);
const pkg = _require('./package.json');

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['esm'],
  dts: true,
  clean: true,
  define: {
    __CLI_BUILD__: 'true',
    __CLI_VERSION__: JSON.stringify(pkg.version),
  },
});
