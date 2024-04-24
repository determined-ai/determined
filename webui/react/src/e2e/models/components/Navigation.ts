import { NamedComponent } from 'e2e/models/BaseComponent';
import { NavigationSideBar } from 'e2e/models/components/NavigationSideBar';

/**
 * Returns a representation of the Navigation component.
 * This constructor represents the contents in src/components/Navigation.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Navigation
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class Navigation extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="navigation"]';

  // sidebar is desktop view, tabbar is mobile view
  readonly sidebar = new NavigationSideBar({ parent: this });
  // TODO mobile
  // readonly tabbar = new NavigationTabbar({ parent: this });
}
