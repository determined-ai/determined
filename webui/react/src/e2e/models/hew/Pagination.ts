import { Locator } from '@playwright/test';

import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

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
  readonly perPage: BaseComponent = new BaseComponent({
    parent: this.#options,
    selector: '.ant-pagination-options-size-changer',
  });
  pageButtonLocator(n: number): Locator {
    return this.pwLocator.locator(`.ant-pagination-item.ant-pagination-item-${n}`);
  }
}
