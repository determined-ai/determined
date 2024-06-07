import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Card } from 'e2e/models/hew/Card';

import { ProjectActionDropdown } from './ProjcetActionDropdown';
import { ProjectCreateModal } from './ProjectCreateModal';
import { ProjectDeleteModal } from './ProjectDeleteModal';

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

class ProjectsCard extends Card {
  override readonly actionMenu = new ProjectActionDropdown({
    childNode: new BaseComponent({
      parent: this,
      selector: Card.actionMenuSelector,
    }),
    root: this.root,
  });
}
