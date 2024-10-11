import {
  BaseComponent,
  ComponentContainer,
  NamedComponent,
} from 'playwright-page-model-base/BaseComponent';
import { BasePage } from 'playwright-page-model-base/BasePage';

import { DropdownContent } from 'e2e/models/common/hew/Dropdown';
import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the MultiSortMenu component in src/components/MultiSortMenu.tsx
 */
export class MultiSortMenu extends DropdownContent {
  /**
   * Constructs a MultiSortMenu
   * @param {object} obj
   * @param {ComponentContainer} obj.parent - parent component
   * @param {BasePage} obj.root - root page
   */
  constructor({ parent, root }: { parent: ComponentContainer; root: BasePage }) {
    super({
      clickThisComponentToOpen: new BaseComponent({
        parent,
        selector: '[data-testid="sort-menu-button"]',
      }),
      root,
    });
  }
  readonly multiSort = new MultiSort({ parent: this });
}

/**
 * Represents the MultiSort in the MultiSortMenu component
 */
class MultiSort extends NamedComponent {
  readonly defaultSelector = '[data-test-component="multiSort"]';
  readonly add = new BaseComponent({ parent: this, selector: '[data-test="add"]' });
  readonly reset = new BaseComponent({ parent: this, selector: '[data-test="reset"]' });
  readonly rowContainer = new BaseComponent({ parent: this, selector: '[data-test="rows"]' });
  readonly rows = new MultiSortRow({ parent: this.rowContainer });
}

/**
 * Represents the MultiSortRow in the MultiSortMenu component
 */
class MultiSortRow extends NamedComponent {
  readonly defaultSelector = '[data-test-component="multiSortRow"]';
  readonly column = new ColumnOptions({ parent: this, selector: '[data-test="column"]' });
  readonly order = new DirectionOptions({ parent: this, selector: '[data-test="direction"]' });
  readonly remove = new BaseComponent({ parent: this, selector: '[data-test="remove"]' });
}

/**
 * Represents the ColumnOptions in the MultiSortMenu component
 */
class ColumnOptions extends Select {}

/**
 * Represents the DirectionOptions in the MultiSortMenu component
 */
class DirectionOptions extends Select {}
