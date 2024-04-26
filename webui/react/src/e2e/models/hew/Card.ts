import { BaseComponent } from 'e2e/models/BaseComponent';
import { WorkspaceActionDropdown } from 'e2e/models/components/WorkspaceActionDropdown';

/**
 * Returns a representation of the card component from Hew.
 * This constructor represents the contents in hew/src/kit/Card.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this card
 */
export class Card extends BaseComponent {
  readonly actionMenu = new WorkspaceActionDropdown({
    parent: this,
    selector: '[aria-label="Action menu"]',
  });
}
