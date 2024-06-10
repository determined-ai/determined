import { BaseComponent, CanBeParent, NamedComponent } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';
import { DropdownContent } from 'e2e/models/hew/Dropdown';
import { Message } from 'e2e/models/hew/Message';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Represents the ColumnPickerMenu component in src/components/ColumnPickerMenu.tsx
 */
export class ColumnPickerMenu extends DropdownContent {
  /**
   * Constructs a ColumnPickerMenu
   * @param {object} obj
   * @param {CanBeParent} obj.parent - parent component
   * @param {BasePage} obj.root - root page
   */
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

/**
 * Represents the ColumnPickerTab in the ColumnPickerMenu component
 */
class ColumnPickerTab extends NamedComponent {
  readonly defaultSelector = '[data-test-component="columnPickerTab"]:visible';
  readonly search = new BaseComponent({ parent: this, selector: '[data-test="search"]' });
  readonly columns = new List({ parent: this });
  readonly noResults = new Message({ parent: this.columns });
  readonly showAll = new BaseComponent({ parent: this, selector: '[data-test="showAll"]' });
  readonly reset = new BaseComponent({ parent: this, selector: '[data-test="reset"]' });
}

/**
 * Represents the List in the ColumnPickerMenu component
 */
class List extends NamedComponent {
  readonly defaultSelector = '[data-test="columns"]';
  readonly rows = new Row({ parent: this });

  /**
   * Returns a representation of a list row with the specified testid.
   * @param {string} [testid] - the testid of the tab, generally the name
   */
  public listItem(testid: string): Row {
    return new Row({
      attachment: `[${this.rows.keyAttribute}="${testid}"]`,
      parent: this,
    });
  }

  /**
   * Returns a list of keys associated with attributes from rows from the entire table.
   */
  async allRowKeys(): Promise<string[]> {
    const { pwLocator, keyAttribute } = this.rows;
    const rows = await pwLocator.all();
    return Promise.all(
      rows.map(async (row) => {
        return (
          (await row.getAttribute(keyAttribute)) ||
          Promise.reject(new Error(`All rows should have the attribute ${keyAttribute}`))
        );
      }),
    );
  }
}

/**
 * Represents a Row in the ColumnPickerMenu component
 */
class Row extends NamedComponent {
  readonly defaultSelector = '[data-test="row"]';
  readonly keyAttribute = 'data-test-id';
  readonly checkbox = new BaseComponent({ parent: this, selector: '[data-test="checkbox"]' });
}
