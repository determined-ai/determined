import { BaseComponent, BaseReactFragment } from 'e2e/models/BaseComponent';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the NavigationSideBar component.
 * This constructor represents the contents in src/components/NavigationSideBar.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this NavigationSideBar
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class NavigationSideBar extends BaseReactFragment {
  readonly #nav: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="navSidebar"]',
  });
  readonly headerDropdown: HeaderDropdown = new HeaderDropdown({
    parent: this.#nav,
    selector: '[data-testid="headerDropdown"]',
  });
  // TODO the rest of the sidebar items
  // TODO nameplate with parent = this.headerDropdown
  // TODO UserSettings works as a drawer on desktop view after clicking on nav.headerDropdown.settings
  // TODO readonly userSettings: UserSettings = new UserSettings({ parent: this });
}

class HeaderDropdown extends Dropdown {
  readonly admin: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('admin'),
  });
  readonly settings: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('settings'),
  });
  readonly theme: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('theme-toggle'),
  });
  readonly signOut: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('sign-out'),
  });
}
