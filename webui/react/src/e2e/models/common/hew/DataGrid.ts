import {
  BaseComponent,
  NamedComponent,
  NamedComponentArgs,
} from 'playwright-page-model-base/BaseComponent';

import { expect } from 'e2e/fixtures/global-fixtures';
import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { printMap } from 'e2e/utils/debug';

class IndexNotFoundError extends Error {}
class NameNotFoundError extends Error {}

type RowClass<RowType extends Row<HeadRowType>, HeadRowType extends HeadRow<RowType>> = new (
  args: RowArgs<RowType, HeadRowType>,
) => RowType;
type HeadRowClass<RowType extends Row<HeadRowType>, HeadRowType extends HeadRow<RowType>> = new (
  args: HeadRowArgs<RowType, HeadRowType>,
) => HeadRowType;

export type RowArgs<
  RowType extends Row<HeadRowType>,
  HeadRowType extends HeadRow<RowType>,
> = NamedComponentArgs & { parentTable: DataGrid<RowType, HeadRowType> };
export type HeadRowArgs<
  RowType extends Row<HeadRowType>,
  HeadRowType extends HeadRow<RowType>,
> = NamedComponentArgs & { parentTable: DataGrid<RowType, HeadRowType> };
export type TableArgs<
  RowType extends Row<HeadRowType>,
  HeadRowType extends HeadRow<RowType>,
> = NamedComponentArgs & {
  rowType: RowClass<RowType, HeadRowType>;
  headRowType: HeadRowClass<RowType, HeadRowType>;
};

/**
 * Represents the DataGrid component in hew/src/kit/DataGrid.tsx
 */
export class DataGrid<
  RowType extends Row<HeadRowType>,
  HeadRowType extends HeadRow<RowType>,
> extends NamedComponent {
  static readonly columnIndexAttribute = 'aria-colindex';
  readonly defaultSelector = '[class^="DataGrid_base"]';

  /**
   * Constructs a DataGrid
   * @param {object} args
   * @param {RowType} args.rowType - Value for the RowType used to instanciate rows
   * @param {HeadRowType} args.headRowType - Value of the HeadRowType used to instanciate the head row
   */
  constructor(args: TableArgs<RowType, HeadRowType>) {
    super(args);
    this.#rowType = args.rowType;
    this.rows = new args.rowType({
      parent: this.#body,
      parentTable: this,
    });
    this.headRow = new args.headRowType({
      parent: this.#head,
      parentTable: this,
    });
  }
  readonly canvasTable = new BaseComponent({
    parent: this,
    selector: 'canvas[data-testid="data-grid-canvas"] table',
  });
  readonly #otherCanvas = new BaseComponent({
    parent: this,
    selector: 'canvas:not([data-testid])',
  });
  #columnheight?: number;
  readonly #rowType: RowClass<RowType, HeadRowType>;
  readonly rows: RowType;
  readonly headRow: HeadRowType;
  readonly #body = new BaseComponent({
    parent: this.canvasTable,
    selector: 'tbody',
  });
  readonly #head = new BaseComponent({
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
   * Scolls the table
   * @param {object} obj
   * @param {number} [obj.xAbsolute] - The x coordinate. If not provided, the table will incrementally scroll to the right.
   */
  private async scrollTable({ xAbsolute }: { xAbsolute?: number } = {}): Promise<void> {
    const box = await this.pwLocator.boundingBox();
    if (box === null) {
      throw new Error('Expected to see a bounding box for the table.');
    }
    const page = this.root._page;
    // move mouse to the center of the table
    await page.mouse.move((box.x + box.width) / 2, (box.y + box.height) / 2);
    // scroll the table to the right by the width of the table minus 500
    // All the permanent columns on the left together are under 500px wide
    await page.mouse.wheel(xAbsolute ? xAbsolute : box.width - 500, 0);
    // wait for the scroll to happen
    await page.waitForTimeout(2_000);
  }

  /**
   * Scrolls the table to the left
   */
  async scrollLeft(): Promise<void> {
    await this.scrollTable({ xAbsolute: -9999 });
  }

  /**
   * Increments the scroll of the table to the right
   */
  async incrementScrollGenerator(): Promise<() => Promise<boolean>> {
    await this.scrollLeft();

    let prevIndexes: (string | null)[] = [];
    const incrementScroll = async () => {
      await this.scrollTable();
      const cells = await this.headRow.cells.pwLocator.all();
      const indexes = await Promise.all(
        cells.map(async (cell) => {
          return await cell.getAttribute(DataGrid.columnIndexAttribute);
        }),
      );
      if (JSON.stringify(indexes) === JSON.stringify(prevIndexes)) {
        return false;
      }
      prevIndexes = indexes;
      return true;
    };
    return incrementScroll;
  }

  /**
   * Scrolls the column into view
   * @param {string} s - The column name
   */
  async scrollColumnIntoViewByName(s: string): Promise<void> {
    try {
      return await this.scrollColumnIntoView(this.headRow.getColumnDef(s));
    } catch (e) {
      if (e instanceof IndexNotFoundError) {
        throw new NameNotFoundError(`Column with name ${s} not found.`);
      }
    }
  }

  /**
   * Scrolls the column into view
   * @param {number} index - The column index
   */
  async scrollColumnIntoView(index: number): Promise<void> {
    if (index === 0) {
      return;
    }
    const incrementScroll = await this.incrementScrollGenerator();

    const checkColumnInView = async () => {
      const cells = await this.headRow.cells.pwLocator.all();
      if (cells.length === 0) {
        throw new Error('Expected to see more than 0 columns.');
      }
      return (
        await Promise.all(
          cells.map(async (cell) => {
            return (
              index + 1 === parseInt((await cell.getAttribute(DataGrid.columnIndexAttribute)) || '')
            );
          }),
        )
      ).some(Boolean);
    };
    do {
      if (await checkColumnInView()) {
        return;
      }
    } while (await incrementScroll());
    throw new IndexNotFoundError(`Column with index ${index} not found.`);
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
          Promise.reject(new Error(`All rows should have the attribute ${indexAttribute}`))
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
        ((await (await row.getCellByColumnName(columnName)).pwLocator.textContent()) || '').indexOf(
          value,
        ) > -1
      );
    });
    if (rows.length !== 1) {
      const names = await Promise.all(
        rows.map(
          async (row) => await (await row.getCellByColumnName('Name')).pwLocator.textContent(),
        ),
      );
      throw new Error(
        `Expected one row to match ${columnName}:${value}. Found ${rows.length} rows that meet the condition: ${names}.`,
      );
    }
    return rows[0];
  }
}

/**
 * Represents a row from the DataGrid component
 */
export class Row<HeadRowType extends HeadRow<Row<HeadRowType>>> extends NamedComponent {
  readonly defaultSelector = 'tr';
  readonly indexAttribute = 'aria-rowindex';

  /**
   * Constructs a Row
   * @param {object} args
   * @param {ComponentContainer} args.parent - The parent used to locate this Row
   * @param {string} args.selector - Used as a selector uesd to locate this object
   * @param {DataGrid<RowType, HeadRowType>} [args.parentTable] - Reference to the original table
   */
  constructor(args: RowArgs<Row<HeadRowType>, HeadRowType>) {
    super(args);
    this.parentTable = args.parentTable;
  }
  parentTable: DataGrid<Row<HeadRowType>, HeadRowType>;

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
    // (index + 2) here to account for header row and counting from 0
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
      timeout: 10_000,
    });
  }

  /**
   * Returns a cell from an index. Start counting at 0.
   */
  async getCellByColIndex(n: number): Promise<BaseComponent> {
    await this.parentTable.scrollColumnIntoView(n);
    return new BaseComponent({
      parent: this,
      selector: `[${DataGrid.columnIndexAttribute}="${n + 1}"]`,
    });
  }

  /**
   * Returns a cell from a column name.
   */
  async getCellByColumnName(s: string): Promise<BaseComponent> {
    return await this.getCellByColIndex(this.parentTable.headRow.getColumnDef(s));
  }
}

/**
 * Represents the head row from the DataGrid component
 */
export class HeadRow<RowType extends Row<HeadRow<RowType>>> extends NamedComponent {
  readonly defaultSelector = 'tr';
  readonly parentTable: DataGrid<RowType, HeadRow<RowType>>;

  /**
   * Constructs a HeadRow
   * @param {object} args
   * @param {ComponentContainer} args.parent - The parent used to locate this HeadRow
   * @param {string} args.selector - Used as a selector uesd to locate this object
   */
  constructor(args: HeadRowArgs<RowType, HeadRow<RowType>>) {
    super(args);
    this.parentTable = args.parentTable;
  }

  readonly cells = new BaseComponent({
    parent: this,
    selector: 'th',
  });
  readonly selectDropdown = new HeaderDropdown({
    clickThisComponentToOpen: new BaseComponent({
      parent: this,
      selector: `[${DataGrid.columnIndexAttribute}="1"]`,
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

  getColumnDef(columnName: string): number {
    const index = this.columnDefs.get(columnName);
    if (index === undefined) {
      throw new Error(
        `Column with title ${columnName} expected but not found.\n${printMap(this.#columnDefs)}`,
      );
    }
    return index;
  }

  /**
   * Sets Column Definitions
   * Row.getCellByColumnName will fail without running this first.
   */
  async setColumnDefs(): Promise<Map<string, number>> {
    // make sure we see enough columns before getting textContent of each.
    // there are four columns on the left
    this.#columnDefs.clear();
    await expect.poll(async () => await this.cells.pwLocator.count()).toBeGreaterThanOrEqual(4);

    const setVisibleColumns = async () => {
      const cells = await this.cells.pwLocator.all();
      if (cells.length === 0) {
        throw new Error('Expected to see more than 0 columns.');
      }
      await Promise.all(
        cells.map(async (cell) => {
          const index = await cell.getAttribute(DataGrid.columnIndexAttribute);
          if (index === null)
            throw new Error(
              `All header cells should have the attribute ${DataGrid.columnIndexAttribute}`,
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
          this.#columnDefs.set(text, parseInt(index) - 1);
        }),
      );
    };

    const incrementScroll = await this.parentTable.incrementScrollGenerator();
    do {
      await setVisibleColumns();
      await this.root.browserLog(printMap(this.#columnDefs));
    } while (await incrementScroll());
    return this.#columnDefs;
  }

  /**
   * Clicks the head row's select button
   */
  async clickSelectDropdown(): Promise<void> {
    // magic numbers for the select button
    await this.parentTable.pwLocator.click({ position: { x: 5, y: 5 } });
  }
}

/**
 * Represents the HeaderDropdown from the DataGrid component
 */
class HeaderDropdown extends DropdownMenu {
  readonly select5 = this.menuItem('select-5');
  readonly select10 = this.menuItem('select-10');
  readonly select25 = this.menuItem('select-25');
  readonly selectAll = this.menuItem('select-all');
}
