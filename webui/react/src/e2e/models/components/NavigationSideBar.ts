import { BaseComponent, BaseReactFragment } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the NavigationSideBar component.
 * This constructor represents the contents in src/components/NavigationSideBar.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this NavigationSideBar
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */

export class NavigationSideBar extends BaseReactFragment {
  readonly #nav: BaseComponent = new BaseComponent({ parent: this, selector: "[data-testid='navSidebar']" });
  readonly headerDropdown: BaseComponent = new BaseComponent({ parent: this.#nav, selector: "[data-testid='headerDropdown']" });
  readonly headerDropdownMenu: BaseComponent = new BaseComponent({ parent: this.root, selector: "ul.ant-dropdown-menu" });
  readonly headerDropdownMenuItemsAdmin: BaseComponent = new BaseComponent({ parent: this.headerDropdownMenu, selector: "li.ant-dropdown-menu-item[data-menu-id$='admin']" });
  readonly headerDropdownMenuItemsSettings: BaseComponent = new BaseComponent({ parent: this.headerDropdownMenu, selector: "li.ant-dropdown-menu-item[data-menu-id$='settings']" });
  readonly headerDropdownMenuItemsTheme: BaseComponent = new BaseComponent({ parent: this.headerDropdownMenu, selector: "li.ant-dropdown-menu-item[data-menu-id$='theme-toggle']" });
  readonly headerDropdownMenuItemsSignOut: BaseComponent = new BaseComponent({ parent: this.headerDropdownMenu, selector: "li.ant-dropdown-menu-item[data-menu-id$='sign-out']" });

  // UserSettings works as a drawer on desktop view after clicking on nav.nameplate.dropdown.settings
  // readonly userSettings: UserSettings = new UserSettings({ parent: this });
}
