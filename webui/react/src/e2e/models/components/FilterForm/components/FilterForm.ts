import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { FilterGroup } from 'e2e/models/components/FilterForm/components/FilterGroup';

/**
 * Returns a representation of the FilterForm component.
 * This constructor represents the contents in src/components/FilterForm.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this FilterForm
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class FilterForm extends NamedComponent {
  readonly defaultSelector = '[data-test-component="FilterForm"]';
  readonly showArchived = new BaseComponent({
    parent: this,
    selector: '[data-test="header"] button',
  });
  readonly filter = new FilterGroup({
    parent: this,
  });
  readonly addCondition = new BaseComponent({
    parent: this,
    selector: '[data-test="addCondition"]',
  });
  readonly addConditionGroup = new BaseComponent({
    parent: this,
    selector: '[data-test="addConditionGroup"]',
  });
  readonly clearFilters = new BaseComponent({
    parent: this,
    selector: '[data-test="clearFilters"]',
  });
}
