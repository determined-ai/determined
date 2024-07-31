import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { DatePicker } from 'e2e/models/common/hew/DatePicker';
import { Select } from 'e2e/models/common/hew/Select';
import { ConjunctionContainer } from 'e2e/models/components/FilterForm/components/ConjunctionContainer';

/**
 * Represents the FilterField component in src/components/FilterForm/components/FilterField.tsx
 */
export class FilterField extends NamedComponent {
  readonly defaultSelector = '[data-test-component="FilterField"]';

  readonly conjunctionContainer = new ConjunctionContainer({ parent: this });
  readonly fieldCard = new BaseComponent({ parent: this, selector: '[data-test="fieldCard"]' });
  readonly columnName = new Select({
    parent: this.fieldCard,
    selector: '[data-test="columnName"]',
  });
  readonly operator = new Select({ parent: this.fieldCard, selector: '[data-test="operator"]' });
  readonly valueSpecial = new Select({ parent: this.fieldCard, selector: '[data-test="special"]' });
  readonly valueText = new BaseComponent({
    parent: this.fieldCard,
    selector: '[data-test="text"]',
  });
  readonly valueNumber = new BaseComponent({
    parent: this.fieldCard,
    selector: '[data-test="number"]',
  });
  readonly valueDate = new DatePicker({ parent: this.fieldCard });
  readonly remove = new BaseComponent({ parent: this, selector: '[data-test="remove"]' });
  readonly move = new BaseComponent({ parent: this, selector: '[data-test="move"]' });
}
