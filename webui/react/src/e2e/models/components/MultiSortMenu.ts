import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Dropdown } from 'e2e/models/hew/Dropdown';
import { Select } from 'e2e/models/hew/Select';

/**
 * Returns a representation of the MultiSortMenu component.
 * This constructor represents the contents in src/components/MultiSortMenu.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this MultiSortMenu
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class MultiSortMenu extends NamedComponent {
  readonly defaultSelector = '[data-test-component="multiSortMenu"]';
  readonly dropdown = new Dropdown({
    parent: this._parent,
    selector: 'button' + this.defaultSelector,
  });
  readonly multiSort = new MultiSort({ parent: this.dropdown });
}

class MultiSort extends NamedComponent {
  readonly defaultSelector = '[data-test-component="multiSort"]';
  readonly add = new BaseComponent({ parent: this, selector: '[data-test="add"]' });
  readonly reset = new BaseComponent({ parent: this, selector: '[data-test="reset"]' });
  readonly rows = new MultiSortRow({ parent: this, selector: '[data-test="reset"]' });
}

class MultiSortRow extends NamedComponent {
  readonly defaultSelector = '[data-test-component="multiSortRow"]';
  readonly column = new ColumnOptions({ parent: this, selector: '[data-test="column"]' });
  readonly order = new DirectionOptions({ parent: this, selector: '[data-test="order"]' });
  readonly remove = new BaseComponent({ parent: this, selector: '[data-test="remove"]' });
}

class ColumnOptions extends Select {}

class DirectionOptions extends Select {}
