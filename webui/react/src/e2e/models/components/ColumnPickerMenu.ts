import { BaseComponent, CanBeParent, NamedComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { DropdownContent } from 'e2e/models/hew/Dropdown';
import { Message } from 'e2e/models/hew/Message';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Returns a representation of the ColumnPickerMenu component.
 * This constructor represents the contents in src/components/ColumnPickerMenu.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this ColumnPickerMenu
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class ColumnPickerMenu extends DropdownContent {
  constructor({ parent, root }: { parent: CanBeParent; root: BasePage }) {
    super({
      childNode: new BaseComponent({
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

class ColumnPickerTab extends NamedComponent {
  readonly defaultSelector = '[data-test-component="columnPickerTab"]';
  readonly search = new BaseComponent({ parent: this, selector: '[data-test="search"]' });
  readonly columns = new List({ parent: this });
  readonly noResults = new Message({ parent: this.columns });
  readonly showAll = new BaseComponent({ parent: this, selector: '[data-test="showAll"]' });
  readonly reset = new BaseComponent({ parent: this, selector: '[data-test="reset"]' });
}

class List extends NamedComponent {
  readonly defaultSelector = '[data-test="columns"]';
  readonly rows = new Row({ parent: this, selector: '[data-test="row"]' });
  /**
   * Returns a representation of a list row with the specified testid.
   * @param {string} [testid] - the testid of the tab, generally the name
   */
  public listItem(testid: string): Row {
    return new Row({
      attachment: `[data-test-id="${testid}"]`,
      parent: this,
    });
  }
}

class Row extends NamedComponent {
  readonly defaultSelector = '[data-test="row"]';
  readonly checkbox = new BaseComponent({ parent: this, selector: '[data-test="checkbox"]' });
}
