import { Pagination } from 'e2e/models/ant/Pagination';
import { BaseReactFragment } from 'e2e/models/BaseComponent';
import { ComparisonView } from 'e2e/models/components/ComparisonView';
import { ExperimentActionDropdown } from 'e2e/models/components/ExperimentActionDropdown';
import { TableActionBar } from 'e2e/models/components/TableActionBar';
import { DataGrid, HeadRow, Row, RowArgs } from 'e2e/models/hew/DataGrid';
import { Message } from 'e2e/models/hew/Message';

/**
 * Returns a representation of the F_ExperiementList component.
 * This constructor represents the contents in src/components/F_ExperiementList.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this F_ExperiementList
 */
export class F_ExperiementList extends BaseReactFragment {
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

class ExperimentHeadRow extends HeadRow<ExperimentRow, ExperimentHeadRow> {}
class ExperimentRow extends Row<ExperimentRow, ExperimentHeadRow> {
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
