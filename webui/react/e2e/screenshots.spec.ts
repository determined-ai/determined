import { expect, test } from '@playwright/test';

test.describe('Screenshots', () => {
  const USERNAME = process.env.USER_NAME ?? '';
  const PASSWORD = process.env.PASSWORD ?? '';

  test.describe('Login', () => {
    test('Login', async ({ page }) => {
      await page.goto('./');
      await expect(page).toHaveScreenshot('login-screen.png');
    });
  });

  test.describe('Design Kit', () => {
    test.beforeEach(async ({ page }) => {
      await test.step('login', async () => {
        await page.goto('./design');
        await page.getByPlaceholder('username').fill(USERNAME);
        await page.getByPlaceholder('password').fill(PASSWORD);
        await page.getByRole('button', { name: 'Sign In' }).click();
        // wait until login is fully successful
        await page.getByRole('heading', { name: 'Accordion' }).waitFor();
        await expect(page).toHaveURL(/design/);
      });
    });

    test.afterEach(async ({ page }) => {
      await page.close();
    });

    test('Accordion', async ({ page }) => {
      const accorditionBasic = page.getByRole('button', { name: 'right Title' });
      await expect(accorditionBasic).toHaveScreenshot('accordion-basic-closed.png');
      await accorditionBasic.click();
      await expect(accorditionBasic).toHaveScreenshot('accordion-basic-open.png');
      await expect(page.getByText('Children')).toHaveScreenshot('accordion-basic-open-child.png');
    });
  });
});
