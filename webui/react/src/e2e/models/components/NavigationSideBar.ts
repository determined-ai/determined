import { BaseComponent, BaseReactFragment } from 'e2e/models/BaseComponent';
import { DropdownMenu } from 'e2e/models/hew/Dropdown';
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
  readonly header = new BaseComponent({
    parent: this.#nav,
    selector: '[data-testid="headerDropdown"]',
  });
  readonly headerNameplate = new Nameplate({
    parent: this.header,
  });
  readonly headerDropdown = new HeaderDropdown({
    childNode: this.header,
    root: this.root,
  });
  readonly home = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Home'}"]`,
  });
  readonly uncategorized = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Uncategorized'}"]`,
  });
  readonly modelRegistry = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Model Registry'}"]`,
  });
  readonly tasks = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Tasks'}"]`,
  });
  readonly webhooks = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Webhooks'}"]`,
  });
  readonly cluster = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Cluster'}"]`,
  });
  readonly workspaces = new BaseComponent({
    parent: this.#nav,
    selector: `a[aria-label="${'Workspaces'}"]`,
  });
  readonly createWorkspace = new BaseComponent({
    parent: this.#nav,
    selector: 'span[aria-label="Create workspace"]',
  });
  /**
   * Returns a representation of a sidebar NavigationItem with the specified label.
   * For example, a workspace pinned to the sidebar is accessible through it's label here.
   * @param {string} label - the label of the tab, generally the name
   */
  public sidebarWorkspaceItem(label: string): SidebarWorkspaceItem {
    return new SidebarWorkspaceItem({
      parent: this.#nav,
      selector: `a[aria-label="${label}"]`,
    });
  }
  // TODO UserSettings works as a drawer on desktop view after clicking on nav.headerDropdown.settings
  // TODO readonly userSettings= new UserSettings({ parent: this });
}
/**
 * Returns a representation of the HeaderDropdown component.
 * Until the dropdown component supports test ids, this model will match any open dropdown.
 * This constructor represents the contents in src/components/NavigationSideBar.
 *
 * The dropdown can be opened by calling the open method.
 * @param {object} obj
 * @param {BasePage} obj.root - root of the page
 * @param {ComponentBasics} [obj.childNode] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
 * @param {Function} [obj.openMethod] - optional if `childNode` is present. It's the method to open the dropdown.
 */
class HeaderDropdown extends DropdownMenu {
  readonly admin = this.menuItem('admin');
  readonly settings = this.menuItem('settings');
  readonly theme = this.menuItem('theme-toggle');
  readonly signOut = this.menuItem('sign-out');
}

/**
 * Returns the representation of a SidebarWorkspaceItem.
 * This constructor is a base class for any component in src/components/NavigationSideBar.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this BaseComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class SidebarWorkspaceItem extends BaseComponent {
  readonly actionMenu = new WorkspaceActionDropdown({
    openMethod: async () => {
      await this.pwLocator.click({ button: 'right' });
    },
    root: this.root,
  });
}
