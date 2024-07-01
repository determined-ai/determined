import { BasePage } from 'e2e/models/common/base/BasePage';
import { DynamicTabs } from 'e2e/models/components/DynamicTabs';
import { F_ExperiementList } from 'e2e/models/components/F_ExperiementList';
import { PageComponent } from 'e2e/models/components/Page';

/**
 * Represents the SignIn page from src/pages/ProjectDetails.tsx
 */
export class ProjectDetails extends BasePage {
  readonly title = /Uncategorized Experiments|Project Details/;
  readonly url = /projects\/\d+/;

  /**
   * Visits the project details page.
   * @param {string} [projectID] - The Project to visit. Defaults to '1' for uncategorized
   */
  async gotoProject(projectID: string = '1', args = {}): Promise<this> {
    return await this.goto({ ...args, url: `projects/${projectID}` });
  }

  readonly pageComponent = new PageComponent({ parent: this });
  readonly dynamicTabs = new DynamicTabs({ parent: this.pageComponent });
  readonly runsTab = this.dynamicTabs.pivot.tab('runs');
  readonly experimentsTab = this.dynamicTabs.pivot.tab('experiments');
  readonly searchesTab = this.dynamicTabs.pivot.tab('searches');
  readonly notesTab = this.dynamicTabs.pivot.tab('notes');
  readonly f_experiemntList = new F_ExperiementList({ parent: this.dynamicTabs.pivot.tabContent });
  // TODO add models for other tabs
}
