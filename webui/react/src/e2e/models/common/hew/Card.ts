import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';

/**
 * Represents the Card component in hew/src/kit/Card.tsx
 */
export class Card extends NamedComponent {
  override defaultSelector = '[data-testid^="card"]';
  readonly actionMenuContainer = new BaseComponent({
    parent: this,
    selector: '[aria-label="Action menu"]',
  });
  readonly actionMenu = new DropdownMenu({
    clickThisComponentToOpen: this.actionMenuContainer,
    root: this.root,
  });
}
