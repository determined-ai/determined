import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { Select } from 'e2e/models/hew/Select';

/**
 * Returns a representation of the ConjunctionContainer component.
 * This constructor represents the contents in src/components/ConjunctionContainer.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this ConjunctionContainer
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class ConjunctionContainer extends NamedComponent {
  readonly defaultSelector = '[data-test-component="ConjunctionContainer"]';
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
  readonly and = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('and'),
  });
  readonly or = new BaseComponent({ parent: this._menu, selector: Select.selectorTemplate('or') });
}
