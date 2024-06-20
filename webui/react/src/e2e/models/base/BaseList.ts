import { BaseRow, BaseRowContainer, RowContainerArgs } from './BaseRowContainer';
export { BaseRow };

/**
 * Represents a Base List component
 */
export abstract class BaseList<RowType extends BaseRow> extends BaseRowContainer<RowType> {
  readonly rows: RowType;

  /**
   * Constructs a Base List
   * @param {object} args
   * @param {RowType} args.rowType - Value for the RowType used to instanciate rows
   */
  constructor(args: RowContainerArgs<RowType>) {
    super(args);
    this.rows = new args.rowType({ parent: this });
    this.getRowByDataKey = this.rowByAttributeGenerator(this.rows.keyAttribute);
    this.listItem = this.getRowByDataKey;
  }

  readonly getRowByDataKey: (value: string) => RowType;

  /**
   * Returns a representation of a list row with the specified testid.
   * @param {string} [testid] - the testid of the tab, generally the name
   */
  readonly listItem: (testid: string) => RowType;
}
