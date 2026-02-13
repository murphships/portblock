import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'portblock',
  description: 'mock APIs that actually behave like real ones',
  base: '/portblock/',
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/portblock/logo.svg' }],
  ],
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'Features', link: '/features/smart-fake-data' },
      { text: 'CLI Reference', link: '/cli-reference' },
      { text: 'Examples', link: '/examples' },
      {
        text: 'GitHub',
        link: 'https://github.com/murphships/portblock',
      },
    ],
    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Getting Started', link: '/getting-started' },
          { text: 'Comparison', link: '/comparison' },
        ],
      },
      {
        text: 'Features',
        items: [
          { text: 'Smart Fake Data', link: '/features/smart-fake-data' },
          { text: 'Stateful CRUD', link: '/features/stateful-crud' },
          { text: 'Request Validation', link: '/features/request-validation' },
          { text: 'Prefer Header', link: '/features/prefer-header' },
          { text: 'Query Parameters', link: '/features/query-params' },
          { text: 'Auth Simulation', link: '/features/auth-simulation' },
          { text: 'Proxy Mode', link: '/features/proxy-mode' },
          { text: 'Replay Mode', link: '/features/replay' },
          { text: 'Chaos Mode', link: '/features/chaos-mode' },
          { text: 'Hot Reload', link: '/features/hot-reload' },
          { text: 'Config File', link: '/features/config-file' },
          { text: 'Strict Mode', link: '/features/strict-mode' },
          { text: 'Webhooks', link: '/features/webhooks' },
          { text: 'Test Runner', link: '/features/test-runner' },
          { text: 'Generate', link: '/features/generate' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI Reference', link: '/cli-reference' },
          { text: 'CI/CD', link: '/ci-cd' },
          { text: 'Docker', link: '/docker' },
          { text: 'Examples', link: '/examples' },
        ],
      },
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/murphships/portblock' },
      { icon: 'x', link: 'https://twitter.com/murphships' },
    ],
    footer: {
      message: 'built by <a href="https://twitter.com/murphships">murph</a> — an AI that ships dev tools',
      copyright: 'MIT License',
    },
    search: {
      provider: 'local',
    },
  },
})
