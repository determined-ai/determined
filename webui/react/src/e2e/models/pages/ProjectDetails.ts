import { BaseComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { DynamicTabs } from 'e2e/models/components/DynamicTabs';
import { F_ExperiementList } from 'e2e/models/components/F_ExperiementList';
import { PageComponent } from 'e2e/models/components/Page';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/ProjectDetails.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class ProjectDetails extends BasePage {
  readonly title = new RegExp(
    ProjectDetails.getTitle('(Uncategorized Experiments|Project Details)'),
    'g',
  );
  readonly url: RegExp = /projects\/\d+/;

  /**
   * Returns this so we can chain. Visits the page.
   * ie. await expect(thePage.goto().theElement.pwLocator()).toBeVisible()
   * @param {string} [projectID] - The Project to visit. Defaults to '1' for uncategorized
   */
  async gotoProject(projectID: string = '1', args = {}): Promise<BasePage> {
    return await this.goto({ ...args, url: `projects/${projectID}` });
  }

  readonly pageComponent = new PageComponent({ parent: this });
  readonly dynamicTabs = new DynamicTabs({ parent: this.pageComponent });
  readonly experimentsTab = new BaseComponent({
    parent: this.dynamicTabs.pivot.tablist,
    selector: Pivot.selectorTemplateTabs('experiments'),
  });
  readonly notesTab = new BaseComponent({
    parent: this.dynamicTabs.pivot.tablist,
    selector: Pivot.selectorTemplateTabs('notes'),
  });
  readonly f_experiemntList = new F_ExperiementList({ parent: this.dynamicTabs.pivot.tabContent });
  // TODO add models for ExperimentList
  // TODO add models for notes tab content
}
