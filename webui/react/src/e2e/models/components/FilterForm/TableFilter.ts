import { BaseComponent, CanBeParent } from 'playwright-page-model-base/BaseComponent';

import { BasePage } from 'e2e/models/common/base/BasePage';
import { DropdownContent } from 'e2e/models/common/hew/Dropdown';
import { FilterForm } from 'e2e/models/components/FilterForm/components/FilterForm';

/**
 * Represents the TableFilter component in src/components/FilterForm/TableFilter.tsx
 */
export class TableFilter extends DropdownContent {
  /**
   * Constructs a TableFilter
   * @param {object} obj
   * @param {CanBeParent} obj.parent - parent component
   * @param {BasePage} obj.root - root page
   */
  constructor({ parent, root }: { parent: CanBeParent; root: BasePage }) {
    super({
      clickThisComponentToOpen: new BaseComponent({
        parent,
        selector: '[data-test-component="tableFilter"]',
      }),
      root,
    });
  }
  readonly filterForm = new FilterForm({ parent: this });
}
