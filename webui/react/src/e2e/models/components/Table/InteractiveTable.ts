export { HeadRow, Row } from 'e2e/models/common/ant/Table';
import { NamedComponent, NamedComponentArgs } from 'playwright-page-model-base/BaseComponent';

import { HeadRow, Row, Table } from 'e2e/models/common/ant/Table';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';

type RowClass<RowType> = new (args: NamedComponentArgs) => RowType;
type HeadRowClass<HeadRowType> = new (args: NamedComponentArgs) => HeadRowType;

type InteractiveTableArgs<RowType, HeadRowType> = {
  rowType: RowClass<RowType>;
  headRowType: HeadRowClass<HeadRowType>;
  attachment: string;
};

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
  constructor(
    args: NamedComponentArgs & { tableArgs: InteractiveTableArgs<RowType, HeadRowType> },
  ) {
    super(args);
    this.table = new Table({
      ...args.tableArgs,
      bodySelector: 'tbody.ant-table-tbody',
      headSelector: 'thead.ant-table-thead',
      parent: this,
    });
  }

  readonly table: Table<RowType, HeadRowType>;
  readonly skeleton = new SkeletonTable({ parent: this });
}
