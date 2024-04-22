import { defineConfig, devices } from '@playwright/test';
/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
import * as dotenv from 'dotenv';
dotenv.config();

const serverAddess = process.env.PW_SERVER_ADDRESS
if (serverAddess === undefined) {
  throw new Error('Expected PW_SERVER_ADDRESS to be set.')
}
const port = Number(new URL(serverAddess).port || '3001')

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,

  /* Run tests in files in parallel */
  fullyParallel: !!process.env.CI,

  /* https://playwright.dev/docs/test-timeouts#global-timeout */
  globalTimeout: process.env.PWDEBUG ? 0 : 500_000, // 500s unless debugging
  timeout: 60000, // TODO [INFENG-628] Users page loads slow so we extend 5 minutes and 1 minute per test until we get an isolated backend
  /* Folder for test artifacts such as screenshots, videos, traces, etc. */
  outputDir: './src/e2e/test-results',

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], channel: 'chrome' }
    },

    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },

    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    /* Test against mobile viewports. */
    // {
    //   name: 'Mobile Chrome',
    //   use: { ...devices['Pixel 5'] },
    // },
    // {
    //   name: 'Mobile Safari',
    //   use: { ...devices['iPhone 12'] },
    // },

    /* Test against branded browsers. */
    // {
    //   name: 'edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ..devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],

  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['html', { outputFolder: './src/e2e/playwright-report' }],
    ['junit', { outputFile: './src/e2e/junit-results.xml' }],
    ['list', { printSteps: true }],
  ],

  /* Retry on CI only */
  retries: process.env.CI ? 1 : 0,

  testDir: './src/e2e',

  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: `http://localhost:${port}/`,

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'retain-on-failure',
    video: 'retain-on-failure'
  },

  /* Run your local dev server before starting the tests */
  webServer: {
    command: 'npm run preview',
    port: port,
    reuseExistingServer: !process.env.CI,
  },

  workers: process.env.CI ? 4 : 1,
});
