import globals from 'globals'
import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import reactRecommended from 'eslint-plugin-react/configs/recommended.js'

export default [
  {
    files: ['pcdeditor.js'],
    rules: js.configs.recommended.rules,
    languageOptions: {
      globals: {
        ...globals.node,
        ...globals.browser,
      },
    },
  },
  ...tseslint.configs.recommended,
  reactRecommended,
  {
    languageOptions: {
      ecmaVersion: 12,
      sourceType: 'module',
      parserOptions: {
        ecmaFeatures: {
          jsx: true
        },
      },
    },
  },
  {
    ignores: [
      'wasm_exec.js',
      'pcdeditor.esm.js',
    ],
  },
]
