export { HeadRow, Row } from 'e2e/models/ant/Table';
import { HeadRow, Row, Table, TableArgs } from 'e2e/models/ant/Table';
import { NamedComponent } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

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
    this.table = new Table({ ...args, attachment: '[data-testid="table"]', parent: this });
  }

  readonly table: Table<RowType, HeadRowType>;
  readonly skeleton = new SkeletonTable({ parent: this });
}
