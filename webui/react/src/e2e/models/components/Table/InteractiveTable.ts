import { Locator } from '@playwright/test';

import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

type RowTypeGeneric<RowType> = new ({ parent, selector }: NamedComponentArgs) => RowType;

export type TableArgs<RowType, HeadRowType> = NamedComponentArgs & {
  rowType: RowTypeGeneric<RowType>;
  headRowType: new ({ parent, selector }: NamedComponentArgs) => HeadRowType;
};

/**
 * Returns a representation of the InteractiveTable component.
 * This constructor represents the contents in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this InteractiveTable
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class InteractiveTable<
  RowType extends Row,
  HeadRowType extends HeadRow,
> extends NamedComponent {
  static defaultSelector = 'div[data-test-component="interactiveTable"]';
  constructor({ selector, parent, rowType, headRowType }: TableArgs<RowType, HeadRowType>) {
    super({ parent: parent, selector: selector || InteractiveTable.defaultSelector });
    this.table = new Table({ headRowType, parent: this, rowType });
  }

  readonly table: Table<RowType, HeadRowType>;
  readonly skeleton: SkeletonTable = new SkeletonTable({ parent: this });
}

/**
 * Returns the representation of a Table.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this Table
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class Table<RowType extends Row, HeadRowType extends HeadRow> extends NamedComponent {
  static defaultSelector = '[data-testid="table"]';
  constructor({ parent, selector, rowType, headRowType }: TableArgs<RowType, HeadRowType>) {
    super({ parent: parent, selector: selector || Table.defaultSelector });
    this.#rowType = rowType;
    this.rows = new rowType({ parent: this.#body });
    this.headRow = new headRowType({ parent: this.#head });
    this.getRowByDataKey = this.rowByAttributeGenerator(this.rows.keyAttribute);
  }
  readonly #rowType: RowTypeGeneric<RowType>;
  readonly rows: RowType;
  readonly headRow: HeadRowType;
  readonly #body: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'tbody.ant-table-tbody',
  });
  readonly #head: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'theader.ant-table-thead',
  });
  readonly pagination: Pagination = new Pagination({
    parent: this._parent,
  });

  /**
   * Returns a function that gets a row by an attribute value.
   * @param {string} key - name of the row attribute
   */
  rowByAttributeGenerator(key: string): (value: string) => RowType {
    /**
     * Returns a row by an attribute value.
     * @param {string} value - value of the row attribute
     */
    return (value: string) => {
      // TODO default selector should be instance property to make this easier. We want RowType.defaultSelector
      return new this.#rowType({
        parent: this,
        selector: Row.defaultSelector + `[${key}="${value}"]`,
      });
    };
  }

  readonly getRowByDataKey: (value: string) => RowType;

  /**
   * Returns a list of keys associated with attributes from rows from the entire table.
   */
  async allRowKeys(): Promise<string[]> {
    const keys: string[] = [];
    for (const row of await this.rows.pwLocator.all()) {
      const value = await row.getAttribute(this.rows.keyAttribute);
      if (value === null) {
        throw new Error(`All rows should have the attribute ${this.rows.keyAttribute}`);
      }
      keys.push(value);
    }
    return keys;
  }

  /**
   * Returns a list of new row keys
   * @param {string[]} oldKeys - list of keys to compare the current table against
   */
  async newRowKeys(oldKeys: string[]): Promise<string[]> {
    const newKeys = await this.allRowKeys();
    return newKeys.filter((value) => {
      return oldKeys.indexOf(value) === -1;
    });
  }

  /**
   * Returns a list of rows that match the condition provided
   * @param {(row: RowType) => Promise<boolean>} condition - function which tests each row against a condition
   */
  async filterRows(condition: (row: RowType) => Promise<boolean>): Promise<RowType[]> {
    const filteredRows: RowType[] = [];
    (await this.allRowKeys()).forEach(async (key) => {
      const row = this.getRowByDataKey(key);
      if (await condition(row)) {
        filteredRows.push(row);
      }
    });
    return filteredRows;
  }
}

/**
 * Returns the representation of a Table Row.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this Row
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class Row extends NamedComponent {
  static defaultSelector = 'tr.ant-table-row';
  readonly keyAttribute = 'data-row-key';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || Row.defaultSelector });
  }
  readonly select: BaseComponent = new BaseComponent({
    parent: this,
    selector: '.ant-table-selection-column',
  });

  async getID(): Promise<string> {
    const value = await this.pwLocator.getAttribute(this.keyAttribute);
    if (value === null) {
      throw new Error(`All rows should have the attribute ${this.keyAttribute}`);
    }
    return value;
  }
}

/**
 * Returns the representation of a Table HeadRow.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this HeadRow
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class HeadRow extends NamedComponent {
  static defaultSelector = 'tr';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || HeadRow.defaultSelector });
  }
  readonly selection: BaseComponent = new BaseComponent({
    parent: this,
    selector: '.ant-table-selection-column',
  });
}

/**
 * Returns the representation of a Table Pagination.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this Pagination
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class Pagination extends NamedComponent {
  static defaultSelector = '.ant-pagination';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || Pagination.defaultSelector });
  }
  readonly previous: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-prev',
  });
  readonly next: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-next',
  });
  readonly #options: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'li.ant-pagination-options',
  });
  readonly perPage: BaseComponent = new BaseComponent({
    parent: this.#options,
    selector: '.ant-pagination-options-size-changer',
  });
  pageButtonLocator(n: number): Locator {
    return this.pwLocator.locator(`.ant-pagination-item.ant-pagination-item-${n}`);
  }
}
