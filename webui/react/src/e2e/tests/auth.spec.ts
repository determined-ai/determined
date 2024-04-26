import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { BasePage } from 'e2e/models/BasePage';
import { SignIn } from 'e2e/models/pages/SignIn';

test.describe('Authentication', () => {
  test.beforeEach(async ({ dev }) => {
    await dev.setServerAddress();
  });
  test.afterEach(async ({ page, auth }) => {
    const signInPage = new SignIn(page);
    if ((await page.title()).indexOf(signInPage.title) === -1) {
      await auth.logout();
    }
  });

  test('Login and Logout', async ({ page, auth }) => {
    await test.step('Login', async () => {
      await auth.login();
      await expect(page).toHaveTitle(BasePage.getTitle('Home'));
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
      await auth.login({ waitForURL: /models/ });
      await expect(page).toHaveTitle(BasePage.getTitle('Model Registry'));
    });
  });

  test('Bad credentials should throw an error', async ({ page, auth }) => {
    const signInPage = new SignIn(page);
    await auth.login({ password: 'superstar', username: 'jcom', waitForURL: /login/ });
    await expect(page).toHaveTitle(signInPage.title);
    await expect(page).toHaveURL(/login/);
    await expect(signInPage.detAuth.errors.pwLocator).toBeVisible();
    await expect(signInPage.detAuth.errors.alert.pwLocator).toBeVisible();
    expect(await signInPage.detAuth.errors.message.pwLocator.textContent()).toContain(
      'Login failed',
    );
    expect(await signInPage.detAuth.errors.description.pwLocator.textContent()).toContain(
      'invalid credentials',
    );
  });

  test('Expect submit disabled, and Show multiple errors', async ({ page }) => {
    const signInPage = new SignIn(page);
    await signInPage.goto();
    await expect.soft(signInPage.detAuth.submit.pwLocator).toBeDisabled();
    await signInPage.detAuth.username.pwLocator.fill('chubbs');
    await expect(async () => {
      // on CI, machines are slow and each click can take two seconds!
      await signInPage.detAuth.submit.pwLocator.click();
      await signInPage.detAuth.submit.pwLocator.click();
      // the errors will dissapear after 3 seconds, so let's timeout with 3
      await expect(signInPage.detAuth.errors.close.pwLocator).toHaveCount(2, { timeout: 3000 });
    }).toPass({ timeout: 20000 });
  });
});
