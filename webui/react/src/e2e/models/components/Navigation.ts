import { NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';
import { NavigationSideBar } from 'e2e/models/components/NavigationSideBar';

/**
 * Returns a representation of the Navigation component.
 * This constructor represents the contents in src/components/Navigation.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Navigation
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class Navigation extends NamedComponent {
  static defaultSelector = 'div[data-test-component="navigation"]';
  constructor({ selector, parent }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || Navigation.defaultSelector });
  }

  // sidebar is desktop view, tabbar is mobile view
  readonly sidebar: NavigationSideBar = new NavigationSideBar({ parent: this });
  // TODO mobile
  // readonly tabbar: NavigationTabbar = new NavigationTabbar({ parent: this });
}
