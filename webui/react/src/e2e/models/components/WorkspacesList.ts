import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Card } from 'e2e/models/hew/Card';
import { WorkspaceActionDropdown } from './WorkspaceActionDropdown';

/**
 * Returns a representation of the Workspaces Page component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class WorkspacesList extends NamedComponent {
  override defaultSelector: string = '[id=workspaces]';
  readonly newWorkspaceButton: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="newWorkspace"]',
  });
  readonly cardWithName = (name: string): WorkspaceCard => {
    return Card.withName({parent: this, name: name}, WorkspaceCard)
  };
}

class WorkspaceCard extends Card {
  readonly actionMenu: WorkspaceActionDropdown = new WorkspaceActionDropdown({
    parent: this,
    selector: Card.ActionMenuSelector,
  });
}