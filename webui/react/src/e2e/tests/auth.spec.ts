import { expect, test } from 'e2e/fixtures/global-fixtures';
import { SignIn } from 'e2e/models/pages/SignIn';

test.describe('Authentication', () => {
  test.afterEach(async ({ page, auth }) => {
    const signInPage = new SignIn(page);
    if ((await page.title()).indexOf(signInPage.title) === -1) {
      await auth.logout();
    }
  });

  test('Login and Logout', async ({ page, auth }) => {
    await test.step('Login', async () => {
      await auth.login();
      await expect(page).toHaveDeterminedTitle('Home');
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Logout', async () => {
      const signInPage = new SignIn(page);
      await auth.logout();
      await expect(page).toHaveDeterminedTitle(signInPage.title);
      await expect(page).toHaveURL(/login/);
    });
  });

  test('Login Redirect', async ({ page, auth }) => {
    await test.step('Attempt to Visit a Page', async () => {
      await page.goto('./models');
      await expect(page).toHaveURL(/login/);
    });

    await test.step('Login and Redirect', async () => {
      await auth.login({ expectedURL: /models/ });
      await expect(page).toHaveDeterminedTitle('Model Registry');
    });
  });

  test('Bad Credentials', async ({ page, auth }) => {
    const signInPage = new SignIn(page);
    await auth.login({ expectedURL: /login/, password: 'superstar', username: 'jcom' });
    await expect(page).toHaveDeterminedTitle(signInPage.title);
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

  test('Show Multiple Errors', async ({ page }) => {
    const signInPage = new SignIn(page);
    await signInPage.goto();
    await signInPage.detAuth.username.pwLocator.fill('chubbs');
    await signInPage.detAuth.submit.pwLocator.click();
    await signInPage.detAuth.submit.pwLocator.click();
    await expect(signInPage.detAuth.errors.close.pwLocator).toHaveCount(2);
  });
});
