import {
  BaseComponent,
  ComponentContainer,
  NamedComponent,
} from 'playwright-page-model-base/BaseComponent';
import { BaseList, BaseRow } from 'playwright-page-model-base/BaseList';
import { BasePage } from 'playwright-page-model-base/BasePage';

import { DropdownContent } from 'e2e/models/common/hew/Dropdown';
import { Message } from 'e2e/models/common/hew/Message';
import { Pivot } from 'e2e/models/common/hew/Pivot';

/**
 * Represents the ColumnPickerMenu component in src/components/ColumnPickerMenu.tsx
 */
export class ColumnPickerMenu extends DropdownContent {
  /**
   * Constructs a ColumnPickerMenu
   * @param {object} obj
   * @param {ComponentContainer} obj.parent - parent component
   * @param {BasePage} obj.root - root page
   */
  constructor({ parent, root }: { parent: ComponentContainer; root: BasePage }) {
    super({
      clickThisComponentToOpen: new BaseComponent({
        parent,
        selector: '[data-test-component="columnPickerMenu"]',
      }),
      root,
    });
  }
  readonly pivot = new Pivot({ parent: this });
  readonly generalTab = this.pivot.tab('LOCATION_TYPE_EXPERIMENT');
  readonly metricsTab = this.pivot.tab('LOCATION_TYPE_VALIDATIONS');
  readonly hyperparameterTab = this.pivot.tab('LOCATION_TYPE_HYPERPARAMETERS');
  readonly columnPickerTab = new ColumnPickerTab({ parent: this.pivot.tabContent });
}

/**
 * Represents the ColumnPickerTab in the ColumnPickerMenu component
 */
class ColumnPickerTab extends NamedComponent {
  readonly defaultSelector = '[data-test-component="columnPickerTab"]:visible';
  readonly search = new BaseComponent({ parent: this, selector: '[data-test="search"]' });
  readonly columns = new List({ parent: this, rowType: Row });
  readonly noResults = new Message({ parent: this.columns });
  readonly showAll = new BaseComponent({ parent: this, selector: '[data-test="showAll"]' });
  readonly reset = new BaseComponent({ parent: this, selector: '[data-test="reset"]' });
}

/**
 * Represents the List in the ColumnPickerMenu component
 */
class List extends BaseList<Row> {
  readonly defaultSelector = '[data-test="columns"]';
}

/**
 * Represents a Row in the ColumnPickerMenu component
 */
class Row extends BaseRow {
  readonly defaultSelector = '[data-test="row"]';
  readonly keyAttribute = 'data-test-id';
  readonly checkbox = new BaseComponent({ parent: this, selector: '[data-test="checkbox"]' });
}
