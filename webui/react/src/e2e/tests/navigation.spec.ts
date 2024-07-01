import { expect, test } from 'e2e/fixtures/global-fixtures';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { WorkspaceList } from 'e2e/models/pages/WorkspaceList';

test.describe('Navigation', () => {
  test('Sidebar Navigation', async ({ authedPage }) => {
    // we need any page to access the sidebar, and i haven't modeled the homepage yet
    const userManagementPage = new UserManagement(authedPage);

    await test.step('Login steps', async () => {
      await expect(authedPage).toHaveDeterminedTitle('Home');
      await expect(authedPage).toHaveURL(/dashboard/);
    });

    await test.step('Uncategorized', async () => {
      await userManagementPage.nav.sidebar.uncategorized.pwLocator.click();
      const expectedURL = /projects\/1\/experiments/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveDeterminedTitle('Uncategorized Experiments');
    });

    await test.step('Model Registry', async () => {
      await userManagementPage.nav.sidebar.modelRegistry.pwLocator.click();
      await authedPage.waitForURL(/models/);
      await expect.soft(authedPage).toHaveDeterminedTitle('Model Registry');
    });

    await test.step('Tasks', async () => {
      await userManagementPage.nav.sidebar.tasks.pwLocator.click();
      const expectedURL = /tasks/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveDeterminedTitle('Tasks');
    });

    await test.step('Webhooks', async () => {
      await userManagementPage.nav.sidebar.webhooks.pwLocator.click();
      const expectedURL = /webhooks/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveDeterminedTitle('Webhooks');
    });

    await test.step('Cluster', async () => {
      await userManagementPage.nav.sidebar.cluster.pwLocator.click();
      const expectedURL = /clusters/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveDeterminedTitle('Cluster');
    });

    await test.step('Workspaces', async () => {
      const workspacesPage = new WorkspaceList(authedPage);
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await workspacesPage.waitForURL();
      await expect.soft(authedPage).toHaveDeterminedTitle(workspacesPage.title);
    });

    await test.step('Admin', async () => {
      const userManagementPage = new UserManagement(authedPage);
      await (await userManagementPage.nav.sidebar.headerDropdown.open()).admin.pwLocator.click();
      await userManagementPage.waitForURL();
      await expect.soft(authedPage).toHaveDeterminedTitle(userManagementPage.title);
    });
  });
});
