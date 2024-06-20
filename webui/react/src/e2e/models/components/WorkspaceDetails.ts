import { NamedComponent } from 'e2e/models/base/BaseComponent';
import { Pivot } from 'e2e/models/hew/Pivot';

import { ModelRegistry } from './ModelRegistry';
import { ProjectsComponent } from './ProjectsPage';
import { ResourcePoolsComponent } from './ResourcePoolsPage';
import { TasksComponent } from './TasksPage';

/**
 * Represents the WorkspaceDetails component in src/components/WorkspaceDetails.tsx
 */
export class WorkspaceDetails extends NamedComponent {
  readonly defaultSelector: string = '[id=workspaceDetails]';
  // The details sections are all subpages wrapped with a Pivot tab
  readonly pivot = new Pivot({ parent: this });
  readonly projects = this.pivot.typedTab('projects', ProjectsComponent);
  readonly tasks = this.pivot.typedTab('tasks', TasksComponent);
  readonly modelRegistry = this.pivot.typedTab('models', ModelRegistry);
  readonly resourcePools = this.pivot.typedTab('pools', ResourcePoolsComponent);
}
