import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the card component from Hew.
 * This constructor represents the contents in hew/src/kit/Card.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this card
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Card extends NamedComponent {
  override defaultSelector: string = ''; // must be provided

  readonly actionMenu: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[aria-label="Action menu"]',
  });
  readonly deleteAction: BaseComponent = new BaseComponent({ // DNJ TODO encapsulate in menu component for sidebar
    parent: this.root,
    selector: 'li[data-menu-id$="-delete"]',
  });
}