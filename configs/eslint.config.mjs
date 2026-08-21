// Phase 10 should import this baseline from web/eslint.config.mjs and add the
// Next.js plugin matching the pinned frontend dependencies.
export default [
  {
    ignores: ['node_modules/**', '.next/**', 'out/**', 'coverage/**', 'gen/**'],
  },
  {
    files: ['**/*.{js,mjs,cjs,ts,tsx}'],
    linterOptions: { reportUnusedDisableDirectives: 'error' },
    rules: {
      'no-debugger': 'error',
      'no-duplicate-imports': 'error',
    },
  },
];
