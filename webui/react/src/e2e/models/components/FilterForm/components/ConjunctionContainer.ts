import { BaseComponent, BaseReactFragment } from 'e2e/models/BaseComponent';
import { Select } from 'e2e/models/hew/Select';

/**
 * Returns a representation of the ConjunctionContainer component.
 * This constructor represents the contents in src/components/ConjunctionContainer.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this ConjunctionContainer
 * @param {string} obj.selector - Used instead of `defaultSelector`
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

class ConjunctionSelect extends Select {
  readonly and = this.menuItem('and');
  readonly or = this.menuItem('or');
}
