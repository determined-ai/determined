import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Pagination } from 'e2e/models/common/ant/Pagination';
import { DataGrid, HeadRow, Row, RowArgs } from 'e2e/models/common/hew/DataGrid';
import { Message } from 'e2e/models/common/hew/Message';
import { ComparisonView } from 'e2e/models/components/ComparisonView';
import { ExperimentActionDropdown } from 'e2e/models/components/ExperimentActionDropdown';
import { TableActionBar } from 'e2e/models/components/TableActionBar';

/**
 * Represents the F_ExperimentList component in src/components/F_ExperimentList.tsx
 */
export class F_ExperimentList extends BaseReactFragment {
  readonly tableActionBar = new TableActionBar({ parent: this });
  readonly noExperimentsMessage = new Message({ parent: this });
  // TODO no filtered experiments
  // TODO error
  readonly comparisonView = new ComparisonView({ parent: this });
  readonly dataGrid = new DataGrid({
    headRowType: ExperimentHeadRow,
    parent: this.comparisonView.initial,
    rowType: ExperimentRow,
  });
  readonly pagination = new Pagination({ parent: this });
}

/**
 * Represents the ExperimentHeadRow in the F_ExperimentList component
 */
class ExperimentHeadRow extends HeadRow<ExperimentRow> {}

/**
 * Represents the ExperimentRow in the F_ExperimentList component
 */
class ExperimentRow extends Row<ExperimentHeadRow> {
  constructor(args: RowArgs<ExperimentRow, ExperimentHeadRow>) {
    super(args);
    this.columnPositions.set('ID', 50);
  }
  readonly experimentActionDropdown = new ExperimentActionDropdown({
    // without bind, we fail on `this.parentTable`
    openMethod: this.rightClick.bind(this),
    root: this.root,
  });
}
