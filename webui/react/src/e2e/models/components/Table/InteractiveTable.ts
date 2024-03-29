import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

export type TableArgs<RowType, HeadRowType> = NamedComponentArgs & {
  rowType: new ({ parent, selector }: NamedComponentArgs) => RowType;
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
 * @param {parentTypes} obj.parent - The parent used to locate this BaseComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class Table<RowType extends Row, HeadRowType extends HeadRow> extends NamedComponent {
  static defaultSelector = 'data-testid="table"';
  constructor({ parent, selector, rowType, headRowType }: TableArgs<RowType, HeadRowType>) {
    super({ parent: parent, selector: selector || Table.defaultSelector });
    this.rows = new rowType({ parent: this.#body });
    this.headRow = new headRowType({ parent: this.#head });
  }
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
}

export class Row extends NamedComponent {
  static defaultSelector = 'tr.ant-table-row';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || Row.defaultSelector });
  }
}

export class HeadRow extends NamedComponent {
  static defaultSelector = 'tr';
  constructor({ parent, selector }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || HeadRow.defaultSelector });
  }
}
