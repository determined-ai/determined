import { NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

export type TableArgs<RowType> = NamedComponentArgs & { rowType: new ({ parent, selector }: NamedComponentArgs) => RowType };

/**
 * Returns a representation of the InteractiveTable component.
 * This constructor represents the contents in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this InteractiveTable
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class InteractiveTable<RowType extends Row> extends NamedComponent {
  static defaultSelector = `div[data-test-component="interactiveTable"]`;
  constructor({ selector, parent, rowType }: TableArgs<RowType>) {
    super({ parent: parent, selector: selector || InteractiveTable.defaultSelector });
    this.table = new Table({ parent: this, rowType: rowType });
  }

  readonly table: Table<RowType>;
  readonly skeleton: SkeletonTable = new SkeletonTable({ parent: this });
}

/**
 * Returns the representation of a Table.
 * This constructor represents the Table in src/components/Table/InteractiveTable.tsx.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this BaseComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
*/
class Table<RowType extends Row> extends NamedComponent {
  static defaultSelector = `data-testid="table"`;
  constructor({ parent, selector, rowType }: TableArgs<RowType> ) {
    super({parent: parent, selector: selector || Table.defaultSelector})
    this.#rowType = rowType
  }
  readonly #rowType: new ({ parent, selector}: NamedComponentArgs) => RowType
  getRows(): RowType {
    return new this.#rowType({parent: this});
  }
}

export class Row extends NamedComponent {
  static defaultSelector = `tr.ant-table-row`;
  constructor({ parent, selector }: NamedComponentArgs ) {
    super({parent: parent, selector: selector || Row.defaultSelector})
  }
}
