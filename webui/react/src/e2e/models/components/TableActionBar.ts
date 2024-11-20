import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { ColumnPickerMenu } from 'e2e/models/components/ColumnPickerMenu';
import { TableFilter } from 'e2e/models/components/FilterForm/TableFilter';
import { MultiSortMenu } from 'e2e/models/components/MultiSortMenu';
import { OptionsMenu } from 'e2e/models/components/OptionsMenu';

/**
 * Represents the TableActionBar component in src/components/TableActionBar.tsx
 */
export class TableActionBar extends NamedComponent {
  defaultSelector = '[data-test-component="tableActionBar"]';
  tableFilter = new TableFilter({ parent: this, root: this.root });
  multiSortMenu = new MultiSortMenu({ parent: this, root: this.root });
  columnPickerMenu = new ColumnPickerMenu({ parent: this, root: this.root });
  optionsMenu = new OptionsMenu({ parent: this, root: this.root });
  actions = new ActionsDropdown({
    clickThisComponentToOpen: new BaseComponent({
      parent: this,
      selector: '[data-test="actionsDropdown"]',
    }),
    root: this.root,
  });
  count = new BaseComponent({ parent: this, selector: '[data-test="count"]' });
  heatmapToggle = new BaseComponent({ parent: this, selector: '[data-test="heatmapToggle"]' });
  compare = new BaseComponent({ parent: this, selector: '[data-test="compare"]' });
  // TODO a bunch of modals
}

/**
 * Represents the ActionsDropdown in the TableActionBar component
 */
class ActionsDropdown extends DropdownMenu {
  readonly openTensorBoard = this.menuItem('View in TensorBoard');
  readonly move = this.menuItem('Move');
  readonly retainLogs = this.menuItem('Retain Logs');
  readonly archive = this.menuItem('Archive');
  readonly uarchive = this.menuItem('Unarchive');
  readonly delete = this.menuItem('Delete');
  readonly activate = this.menuItem('Resume');
  readonly pause = this.menuItem('Pause');
  readonly cancel = this.menuItem('Stop');
  readonly kill = this.menuItem('Kill');
}
