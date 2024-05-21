import {
  BaseComponent,
  NamedComponent,
} from 'e2e/models/BaseComponent';
import { Card } from 'e2e/models/hew/Card';

import { WorkspaceActionDropdown } from './WorkspaceActionDropdown';

/**
 * Returns a representation of the Workspaces Page component.
 * This constructor represents the contents in src/components/WorkspacesList.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class WorkspacesList extends NamedComponent {
  readonly defaultSelector: string = '[id=workspaces]';
  readonly newWorkspaceButton = new BaseComponent({
    parent: this,
    selector: '[data-testid="newWorkspace"]',
  });
  readonly cardWithName = (name: string): WorkspaceCard => {
    return Card.withName({ name: name, parent: this }, WorkspaceCard);
  };
}

/**
 * Returns the representation of a Card.
 * This constructor is a base class for any component in src/components/WorkspacesList.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Card
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class WorkspaceCard extends Card {
  override readonly actionMenu = new WorkspaceActionDropdown({
    childNode: new BaseComponent({
      parent: this,
      selector: Card.actionMenuSelector,
    }),
    root: this.root,
  });
}
