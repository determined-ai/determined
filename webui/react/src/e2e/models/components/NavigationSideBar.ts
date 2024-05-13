import { BaseComponent, BaseReactFragment } from 'e2e/models/BaseComponent';
import { Dropdown } from 'e2e/models/hew/Dropdown';
import { Nameplate } from 'e2e/models/hew/Nameplate';

import { WorkspaceActionDropdown } from './WorkspaceActionDropdown';

/**
 * Returns a representation of the NavigationSideBar component.
 * This constructor represents the contents in src/components/NavigationSideBar.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this NavigationSideBar
 */
export class NavigationSideBar extends BaseReactFragment {
  readonly #nav = new BaseComponent({
    parent: this,
    selector: '[data-testid="navSidebar"]',
  });
  readonly headerDropdown = new HeaderDropdown({
    parent: this.#nav,
    selector: '[data-testid="headerDropdown"]',
  });
  readonly home = this.sidebarItem('Home');
  readonly uncategorized = this.sidebarItem('Uncategorized');
  readonly modelRegistry = this.sidebarItem('Model Registry');
  readonly tasks = this.sidebarItem('Tasks');
  readonly webhooks = this.sidebarItem('Webhooks');
  readonly cluster = this.sidebarItem('Cluster');
  readonly workspaces = this.sidebarItem('Workspaces');
  readonly createWorkspace = new BaseComponent({
    parent: this.#nav,
    selector: 'span[aria-label="Create workspace"]',
  });
  /**
   * Returns a representation of a sidebar NavigationItem with the specified label.
   * For example, a rokspace pinned to the sidebar is accessible through it's label here.
   * @param {string} [label] - the label of the tab, generally the name
   */
  public sidebarItem(label: string): BaseComponent {
    return new BaseComponent({
      parent: this.#nav,
      selector: `a[aria-label="${label}"]`,
    });
  }
  readonly actionMenu = new WorkspaceActionDropdown({
    parent: this.#nav,
    selector: '', // no open-menu button, only right click on sidebar item to open
  });
  // TODO UserSettings works as a drawer on desktop view after clicking on nav.headerDropdown.settings
  // TODO readonly userSettings= new UserSettings({ parent: this });
}

class HeaderDropdown extends Dropdown {
  readonly nameplate = new Nameplate({
    parent: this,
  });
  readonly admin = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('admin'),
  });
  readonly settings = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('settings'),
  });
  readonly theme = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('theme-toggle'),
  });
  readonly signOut = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('sign-out'),
  });
}
