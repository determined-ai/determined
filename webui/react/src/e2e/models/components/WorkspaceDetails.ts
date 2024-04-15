import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Pivot } from '../hew/Pivot';
import { ModelRegistryComponent } from './ModelRegistryPage';
import { ProjectsComponent } from './ProjectsPage';
import { TasksComponent } from './TasksPage';
import { ResourcePoolsComponent } from './ResourcePoolsPage';

/**
 * Returns a representation of the Workspace Details Page component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class WorkspaceDetails extends NamedComponent {
  override defaultSelector: string = '[id=workspaceDetails]';
  // The details sections are all subpages wrapped with a Pivot tab
  readonly projects = new ProjectsPivot({
    parent: this,
  })
  readonly tasks = new TasksPivot({
    parent: this,
  })
  readonly modelRegistry = new ModelsPivot({
    parent: this,
  })
  readonly resourcePools = new PoolsPivot({
    parent: this,
  })
}

class ProjectsPivot extends Pivot {
  readonly tab: BaseComponent = new BaseComponent({
    parent: this,
    selector: Pivot.selectorTemplateTabs('projects'),
  });
  readonly content: ProjectsComponent = new ProjectsComponent({
    parent: this.tabContent,
  })
}

class TasksPivot extends Pivot {
  readonly tab: BaseComponent = new BaseComponent({
    parent: this,
    selector: Pivot.selectorTemplateTabs('tasks'),
  });
  readonly content: TasksComponent = new TasksComponent({
    parent: this.tabContent,
  })
}

class ModelsPivot extends Pivot {
  readonly tab: BaseComponent = new BaseComponent({
    parent: this,
    selector: Pivot.selectorTemplateTabs('models'),
  });
  readonly content: ModelRegistryComponent = new ModelRegistryComponent({
    parent: this.tabContent,
  })
}

class PoolsPivot extends Pivot {
  readonly tab: BaseComponent = new BaseComponent({
    parent: this,
    selector: Pivot.selectorTemplateTabs('pools'),
  });
  readonly content: ResourcePoolsComponent = new ResourcePoolsComponent({
    parent: this.tabContent,
  })
}