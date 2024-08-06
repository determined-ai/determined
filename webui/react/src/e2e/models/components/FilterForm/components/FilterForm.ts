import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { FilterGroup } from 'e2e/models/components/FilterForm/components/FilterGroup';

/**
 * Represents the FilterForm component in src/components/FilterForm/components/FilterForm.tsx
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
