import { NamedComponent } from 'e2e/models/BaseComponent';
import { FilterForm } from 'e2e/models/components/FilterForm/components/FilterForm';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the TableFilter component.
 * This constructor represents the contents in src/components/TableFilter.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this TableFilter
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class TableFilter extends NamedComponent {
  readonly defaultSelector = '[data-test-component="tableFilter"]';
  readonly dropdown = new Dropdown({
    parent: this._parent,
    selector: 'button' + this.defaultSelector,
  });
  readonly filterForm = new FilterForm({ parent: this.dropdown });
}
