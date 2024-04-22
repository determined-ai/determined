import { Locator } from '@playwright/test';

import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Select } from 'e2e/models/hew/Select';

/**
 * Returns the representation of a Table Pagination.
 * This constructor represents the Table in src/components/hew/Pagination.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Pagination
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class Pagination extends NamedComponent {
  readonly defaultSelector = '.ant-pagination';
  readonly previous: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-prev',
  });
  readonly next: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-next',
  });
  readonly #options: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-options',
  });
  readonly perPage: PaginationDropdown = new PaginationDropdown({
    parent: this.#options,
    selector: '.ant-pagination-options-size-changer',
  });
  pageButtonLocator(n: number): Locator {
    return this.pwLocator.locator(`.ant-pagination-item.ant-pagination-item-${n}`);
  }
}

/**
 * Returns the representation of a Table Pagination.
 * This constructor represents the Table in src/components/Table/Table.tsx.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this Pagination
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class PaginationDropdown extends Select {
  readonly perPage10: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('10 / page'),
  });
  readonly perPage20: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('20 / page'),
  });
  readonly perPage50: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('50 / page'),
  });
  readonly perPage100: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('100 / page'),
  });
}
