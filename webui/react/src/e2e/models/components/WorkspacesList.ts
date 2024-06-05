import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Card } from 'e2e/models/hew/Card';

import { WorkspaceActionDropdown } from './WorkspaceActionDropdown';

/**
 * Represents the WorkspacesList component in src/components/WorkspacesList.tsx
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
 * Represents the WorkspaceCard in the WorkspacesList component
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
