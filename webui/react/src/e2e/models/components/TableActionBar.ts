import { NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the TableActionBar component.
 * This constructor represents the contents in src/components/TableActionBar.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this TableActionBar
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class TableActionBar extends NamedComponent {
  defaultSelector = '[data-test-component="tableActionBar"]';
  // TODO - filter, sort, column, menu, actions, heatmap toggle, compare
}
