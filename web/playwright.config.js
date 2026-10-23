import { defineConfig } from '@playwright/test'
export default defineConfig({
 testDir: './tests', timeout: 120000, expect: { timeout: 20000 }, fullyParallel: false, workers: 1,
 use: { baseURL: 'http://127.0.0.1:3137', viewport: { width: 1440, height: 1000 }, screenshot: 'only-on-failure', trace: 'retain-on-failure' },
 projects: ['chromium', 'firefox', 'webkit'].map(browserName => ({ name: browserName, use: { browserName } })),
 webServer: { command: 'node ../scripts/e2e-server.mjs', url: 'http://127.0.0.1:3137/api/status', reuseExistingServer: false, timeout: 120000 },
})
