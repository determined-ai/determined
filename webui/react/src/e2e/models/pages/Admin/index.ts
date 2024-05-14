import { BasePage } from 'e2e/models/BasePage';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Returns a representation of an Admin page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export abstract class AdminPage extends BasePage {
  readonly pivot = new Pivot({ parent: this });
  readonly userTab = this.pivot.tab('user-management');
  readonly groupTab = this.pivot.tab('group-management');
}
