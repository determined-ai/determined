import { BasePage } from 'e2e/models/BasePage';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class UserManagement extends BasePage {
  static title: string | RegExp = 'Determined';
  readonly url: string = 'admin/user-management';
}
