/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
import path from 'path';

import { defineConfig, devices } from '@playwright/test';
import * as dotenv from 'dotenv';

import { baseUrl } from 'e2e/utils/envVars';

dotenv.config({ path: path.resolve(__dirname, '.env') });

const baseURL = baseUrl();
const port = Number(new URL(baseURL).port || 3001);

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,

  /* Run tests in files in parallel */
  fullyParallel: !!process.env.CI,

  /* Folder for test artifacts such as screenshots, videos, traces, etc. */
  outputDir: './src/e2e/test-results',
  /* Folder for test artifacts such as screenshots, videos, traces, etc. */

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], channel: 'chrome' },
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
  timeout: 90_000, // webkit takes longer to run tests

  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    actionTimeout: 5_000,
    baseURL,
    navigationTimeout: 10_000,
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
  },

  /* Run your local dev server before starting the tests */
  webServer: {
    command: 'npm run preview',
    port,
    reuseExistingServer: !process.env.CI,
  },

  workers: process.env.CI ? 3 : undefined,
});
