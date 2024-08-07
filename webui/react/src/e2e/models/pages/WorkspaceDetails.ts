import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { Pivot } from 'e2e/models/common/hew/Pivot';
import { ModelRegistry } from 'e2e/models/components/ModelRegistry';
import { TaskList } from 'e2e/models/components/TaskList';
import { TemplateList } from 'e2e/models/pages/Templates/TemplateList';
import { ResourcePoolsBound } from 'e2e/models/pages/WorkspaceDetails/ResourcePoolsBound';
import { WorkspaceProjects } from 'e2e/models/pages/WorkspaceDetails/WorkspaceProjects';

/**
 * Represents the WorkspaceDetails page from src/pages/WorkspacesDetails.tsx
 */
export class WorkspaceDetails extends DeterminedPage {
  readonly title = '';
  readonly url = /workspaces\/(\d+)\//;

  /**
   * Visits the project details page.
   * @param {string} [workspaceID] - The Workspace to visit. Defaults to '1' for uncategorized
   */
  async gotoWorkspace(workspaceID: number = 1, tab: string = '', args = {}): Promise<this> {
    return await this.goto({ ...args, url: `workspaces/${workspaceID}/${tab}` });
  }

  async getIdFromUrl(): Promise<number> {
    await this._page.waitForURL(this.url);
    const matches = new URL(this._page.url()).pathname.match(this.url);
    if (matches === null) throw new Error('No ID found in the URL');
    return Number(matches[1]);
  }

  readonly pivot = new Pivot({ parent: this });
  readonly projectsTab = this.pivot.tab('projects');
  readonly tasksTab = this.pivot.tab('tasks');
  readonly modelRegistryTab = this.pivot.tab('models');
  readonly resourcePoolsTab = this.pivot.tab('pools');
  readonly workspaceProjects = new WorkspaceProjects({
    parent: this.pivot.tabContent,
  });
  readonly taskList = new TaskList({
    parent: this.pivot.tabContent,
  });
  readonly modelRegistry = new ModelRegistry({
    parent: this.pivot.tabContent,
  });
  readonly resourcePoolsBound = new ResourcePoolsBound({
    parent: this.pivot.tabContent,
  });
  readonly templateList = new TemplateList({
    parent: this.pivot.tabContent,
  });
}
