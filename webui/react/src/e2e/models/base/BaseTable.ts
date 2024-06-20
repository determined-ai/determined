import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/base/BaseComponent';

import { BaseRow, BaseRowContainer } from './BaseRowContainer';
export { BaseRow };

type RowClass<RowType> = new (args: NamedComponentArgs) => RowType;
type HeadRowClass<HeadRowType> = new (args: NamedComponentArgs) => HeadRowType;

export type TableArgs<RowType, HeadRowType> = NamedComponentArgs & {
  rowType: RowClass<RowType>;
  headRowType: HeadRowClass<HeadRowType>;
  bodySelector: string;
  headSelector: string;
};

/**
 * Represents the Table component in src/components/Table/InteractiveTable.tsx.
 */
export abstract class BaseTable<
  RowType extends BaseRow,
  HeadRowType extends BaseHeadRow,
> extends BaseRowContainer<RowType> {
  /**
   * Constructs a Table
   * @param {object} args
   * @param {RowType} args.rowType - Value for the RowType used to instanciate rows
   * @param {HeadRowType} args.headRowType - Value of the HeadRowType used to instanciate the head row
   */
  constructor(args: TableArgs<RowType, HeadRowType>) {
    super(args);
    this._body = new BaseComponent({
      parent: this,
      selector: args.bodySelector,
    });
    this.#head = new BaseComponent({
      parent: this,
      selector: args.headSelector,
    });
    this.rows = new args.rowType({ parent: this._body });
    this.headRow = new args.headRowType({ parent: this.#head });
    this.getRowByDataKey = this.rowByAttributeGenerator(this.rows.keyAttribute);
  }
  readonly rows: RowType;
  readonly headRow: HeadRowType;
  protected readonly _body: BaseComponent;
  readonly #head: BaseComponent;

  readonly getRowByDataKey: (value: string) => RowType;
}

/**
 * Represents the head row from the Table component
 */
export abstract class BaseHeadRow extends NamedComponent {}
