import { BaseComponent, CanBeParent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { FilterForm } from 'e2e/models/components/FilterForm/components/FilterForm';
import { DropdownContent } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the TableFilter component.
 * This constructor represents the contents in src/components/TableFilter.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this TableFilter
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class TableFilter extends DropdownContent {
  constructor({ parent, root }: { parent: CanBeParent; root: BasePage }) {
    super({
      childNode: new BaseComponent({ parent, selector: '[data-test-component="tableFilter"]' }),
      root,
    });
  }
  readonly filterForm = new FilterForm({ parent: this });
}
