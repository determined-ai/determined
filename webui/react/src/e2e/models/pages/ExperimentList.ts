// import { BaseComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class ExperimentList extends BasePage {
  readonly title: RegExp = /\w+ Experiments - Determined/;
  readonly url: RegExp = /projects\/\d+\/experiments/;

  // readonly #actionRow: BaseComponent = new BaseComponent({
  //   parent: this,
  //   selector: '[data-testid="actionRow"]',
  // });

  /**
   * Returns this so we can chain. Visits the page.
   * ie. await expect(thePage.goto().theElement.pwLocator()).toBeVisible()
   * @param {string} [projectID] - The Project to visit. Defaults to '1' for uncategorized
   */
  async gotoProject(projectID: string = '1', args = {}): Promise<BasePage> {
    return await this.goto({...args, url: `projects/${projectID}/experiments`})
  }
}
