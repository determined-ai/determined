import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';
import { Pagination } from 'e2e/models/hew/Pagination';

type RowClass<RowType> = new (args: NamedComponentArgs) => RowType;
type HeadRowClass<HeadRowType> = new (args: NamedComponentArgs) => HeadRowType;

export type TableArgs<RowType, HeadRowType> = NamedComponentArgs & {
  rowType: RowClass<RowType>;
  headRowType: HeadRowClass<HeadRowType>;
};

/**
 * Returns a representation of the InteractiveTable component.
 * This constructor represents the contents in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this InteractiveTable
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 * @param {RowType} [obj.rowType] - Value for the RowType used to instanciate rows
 * @param {HeadRowType} [obj.headRowType] - Value of the HeadRowType used to instanciate the head row
 */
export class InteractiveTable<
  RowType extends Row,
  HeadRowType extends HeadRow,
> extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="interactiveTable"]';
  constructor(args: TableArgs<RowType, HeadRowType>) {
    super(args);
    this.table = new Table({ ...args, parent: this });
  }

  readonly table: Table<RowType, HeadRowType>;
  readonly skeleton: SkeletonTable = new SkeletonTable({ parent: this });
}

/**
 * Returns the representation of a Table.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Table
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 * @param {RowType} [obj.rowType] - Value for the RowType used to instanciate rows
 * @param {HeadRowType} [obj.headRowType] - Value of the HeadRowType used to instanciate the head row
 */
export class Table<RowType extends Row, HeadRowType extends HeadRow> extends NamedComponent {
  readonly defaultSelector = '[data-testid="table"]';
  constructor(args: TableArgs<RowType, HeadRowType>) {
    super(args);
    this.#rowType = args.rowType;
    this.rows = new args.rowType({ parent: this.#body });
    this.headRow = new args.headRowType({ parent: this.#head });
    this.getRowByDataKey = this.rowByAttributeGenerator(this.rows.keyAttribute);
  }
  readonly #rowType: RowClass<RowType>;
  readonly rows: RowType;
  readonly headRow: HeadRowType;
  readonly #body: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'tbody.ant-table-tbody',
  });
  readonly noData: BaseComponent = new BaseComponent({
    parent: this.#body,
    selector: '.ant-empty.ant-empty-normal',
  });
  readonly #head: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'thead.ant-table-thead',
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
      return new this.#rowType({
        attachment: `[${key}="${value}"]`,
        parent: this.#body,
      });
    };
  }

  readonly getRowByDataKey: (value: string) => RowType;

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
    const rowKeys = await this.allRowKeys();
    return (
      await Promise.all(
        rowKeys.map(async (key) => {
          const row = this.getRowByDataKey(key);
          return (await condition(row)) && row;
        }),
      )
    ).filter((c): c is Awaited<RowType> => !!c);
  }
}

/**
 * Returns the representation of a Table Row.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Row
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class Row extends NamedComponent {
  readonly defaultSelector = 'tr.ant-table-row';
  readonly keyAttribute = 'data-row-key';
  readonly select: BaseComponent = new BaseComponent({
    parent: this,
    selector: '.ant-table-selection-column',
  });

  async getId(): Promise<string> {
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
 * @param {CanBeParent} obj.parent - The parent used to locate this HeadRow
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class HeadRow extends NamedComponent {
  readonly defaultSelector = 'tr';
  readonly selectAll: BaseComponent = new BaseComponent({
    parent: this,
    selector: '.ant-table-selection-column .ant-checkbox-input',
  });
}
