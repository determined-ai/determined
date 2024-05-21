import { NamedComponent } from 'e2e/models/BaseComponent';
import { Pivot } from 'e2e/models/hew/Pivot';

import { ModelRegistryPage } from './ModelRegistry';
import { ProjectsComponent } from './ProjectsPage';
import { ResourcePoolsComponent } from './ResourcePoolsPage';
import { TasksComponent } from './TasksPage';

/**
 * Returns a representation of the Workspace Details Page component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class WorkspaceDetails extends NamedComponent {
  readonly defaultSelector: string = '[id=workspaceDetails]';
  // The details sections are all subpages wrapped with a Pivot tab
  readonly pivot = new Pivot({ parent: this });
  readonly projects = this.pivot.typedTab('projects', ProjectsComponent);
  readonly tasks = this.pivot.typedTab('tasks', TasksComponent);
  readonly modelRegistry = this.pivot.typedTab('models', ModelRegistryPage);
  readonly resourcePools = this.pivot.typedTab('pools', ResourcePoolsComponent);
}
