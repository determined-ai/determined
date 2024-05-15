import { expect, Locator } from '@playwright/test';

import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { DropdownMenu } from 'e2e/models/hew/Dropdown';

type RowClass<RowType extends Row<RowType, HeadRowType>, HeadRowType extends HeadRow> = new (
  args: RowArgs<RowType, HeadRowType>,
) => RowType;
type HeadRowClass<HeadRowType> = new (args: HeadRowArgs) => HeadRowType;

export type RowArgs<
  RowType extends Row<RowType, HeadRowType>,
  HeadRowType extends HeadRow,
> = NamedComponentArgs & { parentTable: DataGrid<RowType, HeadRowType> };
export type HeadRowArgs = NamedComponentArgs & { clickableParentLocator: Locator };
export type TableArgs<
  RowType extends Row<RowType, HeadRowType>,
  HeadRowType extends HeadRow,
> = NamedComponentArgs & {
  rowType: RowClass<RowType, HeadRowType>;
  headRowType: HeadRowClass<HeadRowType>;
};

/**
 * Returns a representation of the DataGrid component.
 * This constructor represents the contents in hew/src/kit/DataGrid.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this DataGrid
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 * @param {RowType} [obj.rowType] - Value for the RowType used to instanciate rows
 * @param {HeadRowType} [obj.headRowType] - Value of the HeadRowType used to instanciate the head row
 */
export class DataGrid<
  RowType extends Row<RowType, HeadRowType>,
  HeadRowType extends HeadRow,
> extends NamedComponent {
  readonly defaultSelector: string = '[class^="DataGrid_base"]';
  constructor(args: TableArgs<RowType, HeadRowType>) {
    super(args);
    this.#rowType = args.rowType;
    this.rows = new args.rowType({
      parent: this.#body,
      parentTable: this,
    });
    this.headRow = new args.headRowType({
      clickableParentLocator: this.pwLocator,
      parent: this.#head,
    });
  }
  readonly canvasTable = new BaseComponent({
    parent: this,
    selector: 'canvas[data-testid="data-grid-canvas"] table',
  });
  readonly #otherCanvas: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'canvas:not([data-testid])',
  });
  #columnheight: number | undefined;
  readonly #rowType: RowClass<RowType, HeadRowType>;
  readonly rows: RowType;
  readonly headRow: HeadRowType;
  readonly #body: BaseComponent = new BaseComponent({
    parent: this.canvasTable,
    selector: 'tbody',
  });
  readonly #head: BaseComponent = new BaseComponent({
    parent: this.canvasTable,
    selector: 'thead',
  });

  get columnHeight(): number {
    if (this.#columnheight === undefined || Number.isNaN(this.#columnheight)) {
      throw new Error('Please use setColumnHeight to set the column height');
    }
    return this.#columnheight;
  }

  /**
   * Returns the height of the header column. Trying to click on a row will fail
   * if this method hasn't yet been run.
   */
  async setColumnHeight(): Promise<number> {
    const style = await this.#otherCanvas.pwLocator.getAttribute('style');
    if (style === null) {
      throw new Error("Couldn't find style attribute.");
    }
    const matches = style.match(/height: (\d+)px;/);
    if (matches === null) {
      throw new Error("Couldn't find height in style attribute.");
    }
    this.#columnheight = +matches[1];
    return this.columnHeight;
  }

  /**
   * Returns a row from an index. Start counting at 0.
   */
  getRowByIndex(n: number): RowType {
    return new this.#rowType({
      attachment: `[${this.rows.indexAttribute}="${n + 2}"]`,
      parent: this.#body,
      parentTable: this,
    });
  }

  /**
   * Returns a list of keys associated with attributes from rows from the entire table.
   */
  async allRows(): Promise<string[]> {
    const { pwLocator, indexAttribute } = this.rows;
    const rows = await pwLocator.all();
    return Promise.all(
      rows.map(async (row) => {
        return (
          (await row.getAttribute(indexAttribute)) ||
          Promise.reject(new Error(`all rows should have the attribute ${indexAttribute}`))
        );
      }),
    );
  }

  /**
   * Returns a list of rows that match the condition provided
   * @param {(row: RowType) => Promise<boolean>} condition - function which tests each row against a condition
   */
  async filterRows(condition: (row: RowType) => Promise<boolean>): Promise<RowType[]> {
    return (
      // TODO make sure we're okay if page autorefreshes
      (
        await Promise.all(
          Array.from(Array(await this.rows.pwLocator.count()).keys()).map(async (key) => {
            const row = this.getRowByIndex(key);
            return (await condition(row)) && row;
          }),
        )
      ).filter((c): c is Awaited<RowType> => !!c)
    );
  }

  /**
   * Returns the row which matches the condition. Expects only one match.
   * @param {string} columnName - column to read from
   * @param {string} value - value to match
   */
  async getRowByColumnValue(columnName: string, value: string): Promise<RowType> {
    const rows = await this.filterRows(async (row) => {
      return (
        ((await row.getCellByColumnName(columnName).pwLocator.textContent()) || '').indexOf(value) >
        -1
      );
    });
    if (rows.length !== 1) {
      const names = await Promise.all(
        rows.map(async (row) => await row.getCellByColumnName('Name').pwLocator.textContent()),
      );
      throw new Error(
        `Expected one row to match ${columnName}:${value}. Found ${rows.length} rows that meet the condition: ${names}.`,
      );
    }
    return rows[0];
  }
}

/**
 * Returns the representation of a Table Row.
 * This constructor represents the Table in hew/src/kit/DataGrid.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Row
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 * @param {DataGrid<RowType, HeadRowType>} [obj.parentTable] - Reference to the original table
 */
export class Row<
  RowType extends Row<RowType, HeadRowType>,
  HeadRowType extends HeadRow,
> extends NamedComponent {
  readonly defaultSelector = 'tr';
  readonly indexAttribute = 'aria-rowindex';
  constructor(args: RowArgs<RowType, HeadRowType>) {
    super(args);
    this.parentTable = args.parentTable;
  }
  parentTable: DataGrid<RowType, HeadRowType>;

  protected columnPositions: Map<string, number> = new Map<string, number>([['Select', 5]]);

  /**
   * Returns an aria label value for "selected" status
   */
  async isSelected(): Promise<string | null> {
    return await this.pwLocator.getAttribute('aria-selected');
  }

  /**
   * Returns the index of the row. Start counting at 0.
   */
  async getIndex(): Promise<number> {
    const value = await this.pwLocator.getAttribute(this.indexAttribute);
    if (value === null || Number.isNaN(+value)) {
      throw new Error(`All rows should have the attribute ${this.indexAttribute}`);
    }
    return +value - 2;
  }

  /**
   * Returns ideal Y coordinates for a click
   * @param {object} index - The row's index
   */
  protected getY(index: number): number {
    // (index + 1) here to account for header row
    return (index + 1) * this.parentTable.columnHeight + 5;
  }

  /**
   * Clicks an x coordinate on the row
   * @param {string} columnID - column name
   */
  async clickColumn(columnID: string): Promise<void> {
    const position = this.columnPositions.get(columnID);
    if (position === undefined) {
      throw new Error(
        `We don't know how to click this column. Here are the positions we know: ${JSON.stringify(
          this.columnPositions,
        )}`,
      );
    }
    await this.parentTable.pwLocator.click({
      position: { x: position, y: this.getY(await this.getIndex()) },
    });
  }

  /**
   * Right clicks the row
   */
  async rightClick(): Promise<void> {
    await this.parentTable.pwLocator.click({
      button: 'right',
      position: { x: 5, y: this.getY(await this.getIndex()) },
    });
  }

  /**
   * Returns a cell from an index. Start counting at 0.
   */
  getCellByIndex(n: number): BaseComponent {
    return new BaseComponent({
      parent: this,
      selector: `[aria-colindex="${n + 1}"]`,
    });
  }

  /**
   * Returns a cell from a column name.
   */
  getCellByColumnName(s: string): BaseComponent {
    const map = this.parentTable.headRow.columnDefs;
    const index = map.get(s);
    if (index === undefined) {
      throw new Error(`Column with title expected but not found ${[...map.entries()]}`);
    }
    return this.getCellByIndex(index - 1);
  }
}

/**
 * Returns the representation of a Table HeadRow.
 * This constructor represents the Table in hew/src/kit/DataGrid.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this HeadRow
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class HeadRow extends NamedComponent {
  readonly #columnIndexAttribute = 'aria-colindex';
  readonly defaultSelector = 'tr';
  readonly clickableParentLocator: Locator;
  constructor(args: HeadRowArgs) {
    super(args);
    this.clickableParentLocator = args.clickableParentLocator;
  }

  readonly selectDropdown = new HeaderDropdown({
    childNode: new BaseComponent({
      parent: this,
      selector: `[${this.#columnIndexAttribute}="1"]`,
    }),
    openMethod: this.clickSelectDropdown.bind(this),
    root: this.root,
  });

  #columnDefs = new Map<string, number>();

  get columnDefs(): Map<string, number> {
    if (this.#columnDefs.size === 0) {
      throw new Error('Please set the column definitions using setColumnDefs first!');
    }
    return this.#columnDefs;
  }

  /**
   * Sets Column Definitions
   * Row.getCellByColumnName will fail without running this first.
   */
  async setColumnDefs(): Promise<Map<string, number>> {
    // make sure we see enough columns before getting textContent of each.
    // there are four columns on the left
    await expect
      .poll(async () => await this.pwLocator.locator('th').count())
      .toBeGreaterThanOrEqual(4);
    const cells = await this.pwLocator.locator('th').all();
    if (cells.length === 0) {
      throw new Error('Expected to see more than 0 columns.');
    }
    await Promise.all(
      cells.map(async (cell) => {
        const index = await cell.getAttribute(this.#columnIndexAttribute);
        if (index === null)
          throw new Error(
            `All header cells should have the attribute ${this.#columnIndexAttribute}`,
          );
        if (index !== '1') {
          expect(await cell.textContent()).not.toBe('');
        }
        let text = await cell.textContent();
        if (text === null) {
          if (index === '1') {
            text = '';
          } else {
            throw new Error('Expected to see text in the column header.');
          }
        }
        this.#columnDefs.set(text, parseInt(index));
      }),
    );
    return this.#columnDefs;
  }

  /**
   * Clicks the head row's select button
   */
  async clickSelectDropdown(): Promise<void> {
    // magic numbers for the select button
    await this.clickableParentLocator.click({ position: { x: 5, y: 5 } });
  }
}

/**
 * Returns the representation of the grid's Header Dropdown.
 * Until the dropdown component supports test ids, this model will match any open dropdown.
 * This constructor represents the contents in hew/src/kit/DataGrid.tsx.
 *
 * The dropdown can be opened by calling the open method.
 * @param {object} obj
 * @param {BasePage} obj.root - root of the page
 * @param {ComponentBasics} [obj.childNode] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
 * @param {Function} [obj.openMethod] - optional if `childNode` is present. It's the method to open the dropdown.
 */
class HeaderDropdown extends DropdownMenu {
  readonly select5 = this.menuItem('select-5');
  readonly select10 = this.menuItem('select-10');
  readonly select25 = this.menuItem('select-25');
  readonly selectAll = this.menuItem('select-all');
}
