import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the ConjunctionContainer component in src/components/FilterForm/components/ConjunctionContainer.tsx
 */
export class ConjunctionContainer extends BaseReactFragment {
  readonly where = new BaseComponent({ parent: this, selector: '[data-test="where"]' });
  readonly conjunctionSelect = new ConjunctionSelect({
    parent: this,
    selector: '[data-test="conjunction"]',
  });
  readonly conjunctionContinued = new BaseComponent({
    parent: this,
    selector: '[data-test="conjunctionContinued"]',
  });
}

/**
 * Represents the Select in the ConjunctionContainer component
 */
class ConjunctionSelect extends Select {
  readonly and = this.menuItem('and');
  readonly or = this.menuItem('or');
}
