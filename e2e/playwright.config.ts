import { defineConfig, devices } from '@playwright/test';
import path from 'path';

const headlessEnv = process.env.PLAYWRIGHT_HEADLESS;
const isHeadless = headlessEnv
  ? !['false', '0', 'off'].includes(headlessEnv.toLowerCase())
  : true;

const outputDir = path.resolve(__dirname, 'test-results/artifacts');

export default defineConfig({
  testDir: path.resolve(__dirname, 'src/tests'),
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['list'],
    ['html', { outputFolder: path.resolve(__dirname, 'test-results/html'), open: 'never' }],
  ],
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:4180',
    headless: isHeadless,
    launchOptions: {
      args: ['--disable-crashpad'],
    },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  outputDir,
  timeout: 60_000,
  expect: {
    timeout: 10_000,
  },
});
