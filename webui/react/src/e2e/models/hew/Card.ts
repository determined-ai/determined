import { NamedComponent } from 'e2e/models/BaseComponent';
import { WorkspaceActionDropdown } from 'e2e/models/components/WorkspaceActionDropdown';

/**
 * Returns a representation of the card component from Hew.
 * This constructor represents the contents in hew/src/kit/Card.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this card
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Card extends NamedComponent {
  override defaultSelector: string = ''; // must be provided

  readonly actionMenu = new WorkspaceActionDropdown({
    parent: this,
    selector: '[aria-label="Action menu"]',
  });
}
