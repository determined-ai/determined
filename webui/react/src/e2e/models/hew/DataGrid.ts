import { NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the DataGrid component.
 * This constructor represents the contents in hew/src/kit/DataGrid.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this DataGrid
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class DataGrid extends NamedComponent {
  defaultSelector: string = 'class^="DataGrid_base"'
  // TODO, the same thing I did with interactivetable.ts
  // TODO, method to get row
}

// TODO datagrid row and maybe head row
// TODO, method to click checkbox