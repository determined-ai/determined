import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { ProjectCreateModal } from './ProjectCreateModal';
import { Card } from 'e2e/models/hew/Card';
import { ProjectActionDropdown } from './ProjcetActionDropdown';


/**
 * Returns a representation of the Projects Page component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ProjectsComponent extends NamedComponent {
  override defaultSelector: string = '[id$=projects]';

  readonly newProject = new BaseComponent({
    parent: this,
    selector: '[data-testid=newProject]',
  });
  readonly createProjectModal = new ProjectCreateModal({
    parent: this.root
  });
  readonly cardWithName = (name: string): Card => {
    return Card.withName({ parent: this, name: name }, ProjectsCard)
  };
}

class ProjectsCard extends Card {
  readonly actionMenu: ProjectActionDropdown = new ProjectActionDropdown({
    parent: this,
    selector: Card.ActionMenuSelector,
  });
}
