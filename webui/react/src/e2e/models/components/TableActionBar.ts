import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { ColumnPickerMenu } from 'e2e/models/components/ColumnPickerMenu';
import { TableFilter } from 'e2e/models/components/FilterForm/TableFilter';
import { MultiSortMenu } from 'e2e/models/components/MultiSortMenu';
import { OptionsMenu } from 'e2e/models/components/OptionsMenu';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the TableActionBar component.
 * This constructor represents the contents in src/components/TableActionBar.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this TableActionBar
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class TableActionBar extends NamedComponent {
  defaultSelector = '[data-test-component="tableActionBar"]';
  tableFilter = new TableFilter({ parent: this, selector: '[data-test-component="tableFilter"]' });
  multiSortMenu = new MultiSortMenu({ parent: this });
  columnPickerMenu = new ColumnPickerMenu({ parent: this });
  optionsMenu = new OptionsMenu({ parent: this });
  actions = new ActionsDropdown({ parent: this, selector: '[data-test="actionsDropdown"]' });
  expNum = new BaseComponent({ parent: this, selector: '[data-test="expNum"]' });
  heatmapToggle = new BaseComponent({ parent: this, selector: '[data-test="heatmapToggle"]' });
  compare = new BaseComponent({ parent: this, selector: '[data-test="compare"]' });
  // TODO a bunch of modals
}

class ActionsDropdown extends Dropdown {
  OpenTensorBoard = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('View in TensorBoard'),
  });
  Move = new BaseComponent({ parent: this._menu, selector: Dropdown.selectorTemplate('Move') });
  RetainLogs = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('Retain Logs'),
  });
  Archive = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('Archive'),
  });
  Unarchive = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('Unarchive'),
  });
  Delete = new BaseComponent({ parent: this._menu, selector: Dropdown.selectorTemplate('Delete') });
  Activate = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('Resume'),
  });
  Pause = new BaseComponent({ parent: this._menu, selector: Dropdown.selectorTemplate('Pause') });
  Cancel = new BaseComponent({ parent: this._menu, selector: Dropdown.selectorTemplate('Stop') });
  Kill = new BaseComponent({ parent: this._menu, selector: Dropdown.selectorTemplate('Kill') });
}
