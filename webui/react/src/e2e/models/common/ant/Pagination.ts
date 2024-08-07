import { Locator } from '@playwright/test';
import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the Pagination component from antd/es/pagination/index.d.ts
 */
export class Pagination extends NamedComponent {
  readonly defaultSelector = '.ant-pagination';
  readonly previous = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-prev',
  });
  readonly next = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-next',
  });
  readonly #options = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-options',
  });
  readonly perPage = new PaginationSelect({
    parent: this.#options,
    selector: '.ant-pagination-options-size-changer',
  });
  pageButtonLocator(n: number): Locator {
    return this.pwLocator.locator(`.ant-pagination-item.ant-pagination-item-${n}`);
  }
}

/**
 * Represents the Select in the Pagination component
 */
class PaginationSelect extends Select {
  readonly perPage10 = this.menuItem('10 / page');
  readonly perPage20 = this.menuItem('20 / page');
  readonly perPage50 = this.menuItem('50 / page');
  readonly perPage100 = this.menuItem('100 / page');
}
