import js from '@eslint/js';
import jsdoc from 'eslint-plugin-jsdoc';
import { defineConfig } from 'eslint/config';
import globals from 'globals';
import tseslint from 'typescript-eslint';

/** 源码、测试和维护脚本统一检查；构建产物不参与。 */
export default defineConfig(
  { ignores: ['dist/**', 'node_modules/**', 'coverage/**'] },
  {
    files: ['**/*.{ts,js,mjs,cjs}'],
    languageOptions: { globals: globals.node },
    extends: [js.configs.recommended],
  },
  {
    files: ['**/*.ts'],
    extends: [tseslint.configs.recommended, jsdoc.configs['flat/recommended-typescript-error']],
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', reportUsedIgnorePattern: true },
      ],
    },
  },
  {
    files: ['**/*.{js,mjs,cjs}'],
    extends: [jsdoc.configs['flat/recommended-error']],
  },
  {
    files: ['**/*.{ts,js,mjs,cjs}'],
    rules: {
      // 统一使用单引号；缩进为 2 空格。
      quotes: ['error', 'single', { avoidEscape: true }],
      indent: ['error', 2, { SwitchCase: 1 }],
    },
  },
  {
    files: ['**/*.{ts,js,mjs,cjs}'],
    rules: {
      'jsdoc/require-description': 'error',
      'jsdoc/require-hyphen-before-param-description': ['error', 'always'],
      'jsdoc/check-syntax': 'error',
      'jsdoc/require-asterisk-prefix': 'error',
      'jsdoc/match-description': [
        'error',
        {
          contexts: ['any'],
          matchDescription: '[\\u3400-\\u9fff]',
          message: 'JSDoc 摘要与参数、返回值说明必须包含中文。',
          tags: { param: true, returns: true, property: true },
        },
      ],
      'jsdoc/require-jsdoc': [
        'error',
        { require: { FunctionDeclaration: true, MethodDefinition: true } },
      ],
    },
  },
  {
    files: ['tests/**/*.{ts,mjs}'],
    rules: { 'jsdoc/require-jsdoc': 'off' },
  }
);
