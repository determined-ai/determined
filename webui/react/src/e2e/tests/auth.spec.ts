import { expect, test } from 'e2e/fixtures/global-fixtures';
import { Cluster } from 'e2e/models/pages/Cluster';
import { DefaultRoute } from 'e2e/models/pages/DefaultRoute';
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
      const defaultPage = new DefaultRoute(page);
      await expect(page).toHaveDeterminedTitle(defaultPage.title);
      await expect(page).toHaveURL(defaultPage.url);
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
      await page.goto('./clusters/logs');
      await expect(page).toHaveURL(/login/);
    });

    await test.step('Login and Redirect', async () => {
      await auth.login({ expectedURL: /clusters\/logs/ });
      const clusterPage = new Cluster(page);
      await expect(page).toHaveDeterminedTitle(new RegExp(`${clusterPage.title}( - \\d{1,3}%)?`)); // Cluster page title might contain a percentage value if cluster is active
      await expect(clusterPage.overviewTab.pwLocator).toHaveAttribute('aria-selected', 'false');
      await expect(clusterPage.historicalUsageTab.pwLocator).toHaveAttribute(
        'aria-selected',
        'false',
      );
      await expect(clusterPage.logsTab.pwLocator).toHaveAttribute('aria-selected', 'true');
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
