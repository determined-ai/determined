export { HeadRow, Row } from 'e2e/models/ant/Table';
import { HeadRow, Row, Table, TableArgs } from 'e2e/models/ant/Table';
import { NamedComponent } from 'e2e/models/BaseComponent';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

/**
 * Represents the InteractiveTable component in src/components/Table/InteractiveTable.tsx
 */
export class InteractiveTable<
  RowType extends Row,
  HeadRowType extends HeadRow,
> extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="interactiveTable"]';

  /**
   * Constructor for InteractiveTable
   * @param {object} args
   * @param {RowType} args.rowType - Value for the RowType used to instanciate rows
   * @param {HeadRowType} args.headRowType - Value of the HeadRowType used to instanciate the head row
   */
  constructor(args: TableArgs<RowType, HeadRowType>) {
    super(args);
    this.table = new Table({ ...args, attachment: '[data-testid="table"]', parent: this });
  }

  readonly table: Table<RowType, HeadRowType>;
  readonly skeleton = new SkeletonTable({ parent: this });
}
