import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Card } from 'e2e/models/hew/Card';

import { ProjectActionDropdown } from './ProjcetActionDropdown';
import { ProjectCreateModal } from './ProjectCreateModal';
import { ProjectDeleteModal } from './ProjectDeleteModal';

/**
 * Represents the ProjectsComponent component in src/components/ProjectsComponent.tsx
 */
export class ProjectsComponent extends NamedComponent {
  override defaultSelector: string = '[id$=projects]';
  readonly newProject = new BaseComponent({
    parent: this._parent,
    selector: '[data-testid=newProject]',
  });
  readonly createModal = new ProjectCreateModal({
    parent: this.root,
  });
  readonly deleteModal = new ProjectDeleteModal({
    parent: this.root,
  });
  readonly cardWithName = (name: string): ProjectsCard => {
    return Card.withName({ name: name, parent: this._parent }, ProjectsCard);
  };
}

/**
 * Represents the ProjectsCard in the ProjectsComponent component
 */
class ProjectsCard extends Card {
  override readonly actionMenu = new ProjectActionDropdown({
    childNode: new BaseComponent({
      parent: this,
      selector: Card.actionMenuSelector,
    }),
    root: this.root,
  });
}
