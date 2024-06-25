import { NamedComponent, NamedComponentArgs } from './BaseComponent';

type RowClass<RowType> = new (args: NamedComponentArgs) => RowType;

export type RowContainerArgs<RowType> = NamedComponentArgs & {
  rowType: RowClass<RowType>;
};

/**
 * Represents a Base List component
 */
export abstract class BaseRowContainer<RowType extends BaseRow> extends NamedComponent {
  readonly #rowType: RowClass<RowType>;
  abstract readonly rows: RowType;

  /**
   * Constructs a Base List
   * @param {object} args
   * @param {RowType} args.rowType - Value for the RowType used to instanciate rows
   */
  constructor(args: RowContainerArgs<RowType>) {
    super(args);
    this.#rowType = args.rowType;
  }

  abstract readonly getRowByDataKey: (value: string) => RowType;

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
        parent: this.rows._parent,
      });
    };
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
 * Represents a Base Row component
 */
export abstract class BaseRow extends NamedComponent {
  abstract readonly keyAttribute: string;
  async getId(): Promise<string> {
    const value = await this.pwLocator.getAttribute(this.keyAttribute);
    if (value === null) {
      throw new Error(`All rows should have the attribute ${this.keyAttribute}`);
    }
    return value;
  }
}
