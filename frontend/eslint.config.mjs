import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import eslintPluginAstro from 'eslint-plugin-astro';

export default [
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  ...eslintPluginAstro.configs.recommended,
  {
    ignores: ['dist/', 'node_modules/', '.astro/'],
  },
  // ルールを追加する場合（必要になれば）
  // {
  //   rules:{
  //     // 例: 未使用の変数はエラーではなく警告にする
  //     '@typescript-eslint/no-unused-vars': 'warn',
  //   }
  // }
];
