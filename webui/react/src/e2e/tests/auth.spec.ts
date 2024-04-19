import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { SignIn } from 'e2e/models/pages/SignIn';

test.describe('Authentication', () => {
  test.beforeEach(async ({ dev }) => {
    await dev.setServerAddress();
  });
  test.afterEach(async ({ page, auth }) => {
    const signInPage = new SignIn(page);
    if ((await page.title()) !== signInPage.title) {
      await auth.logout();
    }
  });

  test('Login and Logout', async ({ page, auth }) => {
    await test.step('Login', async () => {
      await auth.login();
      await expect(page).toHaveTitle('Home - (Determined|HPE Machine Learning Development Environment)');
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Logout', async () => {
      const signInPage = new SignIn(page);
      await auth.logout();
      await expect(page).toHaveTitle(signInPage.title);
      await expect(page).toHaveURL(/login/);
    });
  });

  test('Redirect to the target URL after login', async ({ page, auth }) => {
    await test.step('Visit a page and expect redirect back to login', async () => {
      await page.goto('./models');
      await expect(page).toHaveURL(/login/);
    });

    await test.step('Login and expect redirect to previous page', async () => {
      await auth.login(/models/);
      await expect(page).toHaveTitle('Model Registry - (Determined|HPE Machine Learning Development Environment)');
    });
  });

  test('Bad credentials should throw an error', async ({ page, auth }) => {
    const signInPage = new SignIn(page);
    await auth.login(/login/, { password: 'superstar', username: 'jcom' });
    await expect(page).toHaveTitle(signInPage.title);
    await expect(page).toHaveURL(/login/);
    await expect(signInPage.detAuth.errors.pwLocator).toBeVisible();
    expect(await signInPage.detAuth.errors.message.pwLocator.textContent()).toContain(
      'Login failed',
    );
    expect(await signInPage.detAuth.errors.description.pwLocator.textContent()).toContain(
      'invalid credentials',
    );
    await signInPage.detAuth.submit.pwLocator.click();
    await expect(signInPage.detAuth.errors.alert.pwLocator).toHaveCount(2);
    await expect(signInPage.detAuth.errors.message.pwLocator).toHaveCount(2);
  });
});
