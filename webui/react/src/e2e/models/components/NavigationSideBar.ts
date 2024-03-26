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
  readonly headerDropdownMenu: HeaderMenu = new HeaderMenu({ parent: this.root, selector: "ul.ant-dropdown-menu" });
  // UserSettings works as a drawer on desktop view after clicking on nav.nameplate.dropdown.settings
  // readonly userSettings: UserSettings = new UserSettings({ parent: this });
}

class HeaderMenu extends BaseComponent {
  private selectorTemplate(id: string): string {
    return `li.ant-dropdown-menu-item[data-menu-id$='${id}']`
  }
  readonly admin: BaseComponent = new BaseComponent({ parent: this, selector: this.selectorTemplate("admin") });
  readonly settings: BaseComponent = new BaseComponent({ parent: this, selector: this.selectorTemplate("settings") });
  readonly theme: BaseComponent = new BaseComponent({ parent: this, selector: this.selectorTemplate("theme-toggle") });
  readonly signOut: BaseComponent = new BaseComponent({ parent: this, selector: this.selectorTemplate("sign-out") });
}