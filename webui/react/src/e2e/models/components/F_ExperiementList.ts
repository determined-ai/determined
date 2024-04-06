import { BaseReactFragment } from 'e2e/models/BaseComponent';
import { TableActionBar } from 'e2e/models/components/TableActionBar';
import { ComparisonView } from 'e2e/models/components/ComparisonView';
import { ExperimentActionDropdown } from 'e2e/models/components/ExperimentActionDropdown';
import { DataGrid } from 'e2e/models/hew/DataGrid';
import { Pagination } from 'e2e/models/hew/Pagination';

/**
 * Returns a representation of the F_ExperiementList component.
 * This constructor represents the contents in src/components/F_ExperiementList.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this F_ExperiementList
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class F_ExperiementList extends BaseReactFragment {
  readonly tableActionBar: TableActionBar = new TableActionBar({ parent: this })
  // TODO no experiments
  // TODO no filtered experiments
  // TODO error
  readonly comparisonView: ComparisonView = new ComparisonView({ parent: this })
  readonly dataGrid: DataGrid = new DataGrid({ parent: this.comparisonView.initial })
  // There is no button which activates this dropdown. To display it, right-click the grid
  readonly experimentActionDropdown: ExperimentActionDropdown = new ExperimentActionDropdown({ parent: this.root, selector: '' })
  readonly pagination: Pagination = new Pagination({ parent: this })
}

// TODO datagrid row and maybe head row
// TODO, method to click name row