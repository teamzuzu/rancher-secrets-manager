module.exports = {
  root:   true,
  env:    { node: true },
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser:      '@typescript-eslint/parser',
    ecmaVersion: 2020,
    sourceType:  'module',
  },
  extends: ['plugin:vue/vue3-recommended'],
  rules:   {
    'vue/multi-word-component-names': 'off',
    // Rancher extensions mutate `value` in place — this is the framework's pattern
    'vue/no-mutating-props':          'off',
    'vue/max-attributes-per-line':    'off',
  },
};
