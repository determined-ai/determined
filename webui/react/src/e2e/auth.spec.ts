import { expect, test } from '@playwright/test';

test.describe('Authentication', () => {
  const USERNAME = process.env.USER_NAME ?? '';
  const PASSWORD = process.env.PASSWORD ?? '';

  test('Login and Logout', async ({ page }) => {
    await page.goto('/');

    await test.step('Login steps', async () => {
      await page.getByPlaceholder('username').fill(USERNAME);
      await page.getByPlaceholder('password').fill(PASSWORD);
      await page.getByRole('button', { name: 'Sign In' }).click();
      await page.waitForURL(/dashboard/);
      await expect(page).toHaveTitle('Home - Determined');
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Logout steps', async () => {
      await page.locator('header').getByText(USERNAME).click();
      await page.getByRole('link', { name: 'Sign Out' }).click();
      await page.waitForURL(/login/);
      await expect(page).toHaveTitle('Sign In - Determined');
      await expect(page).toHaveURL(/login/);
    });
  });

  test('Redirect to the target URL after login', async ({ page }) => {
    await page.goto('/models');

    await test.step('Login steps', async () => {
      await page.getByPlaceholder('username').fill(USERNAME);
      await page.getByPlaceholder('password').fill(PASSWORD);
      await page.getByRole('button', { name: 'Sign In' }).click();
    });

    await page.waitForURL(/models/);
    await expect(page).toHaveTitle('Model Registry - Determined');
    await expect(page).toHaveURL(/models/);
  });
});
