import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { Nameplate } from 'e2e/models/common/hew/Nameplate';

import { WorkspaceActionDropdown } from './WorkspaceActionDropdown';

/**
 * Represents the NavigationSideBar component in src/components/NavigationSideBar.tsx
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
    clickThisComponentToOpen: this.header,
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
  readonly createWorkspaceFromHover = new BaseComponent({
    parent: this.#nav,
    selector: 'span[aria-label="Create workspace"]',
  });
  // consider this to be a rowContainer
  readonly #pinnedWorkspaces = new BaseComponent({
    parent: this.#nav,
    selector: '[class*="NavigationSideBar_pinnedWorkspaces"]',
  });
  /**
   * Returns a representation of a sidebar NavigationItem with the specified label.
   * For example, a workspace pinned to the sidebar is accessible through it's label here.
   * @param {string} label - the label of the tab, generally the name
   */
  public sidebarWorkspaceItem(label: string): SidebarWorkspaceItem {
    return new SidebarWorkspaceItem({
      parent: this.#pinnedWorkspaces,
      selector: `a[aria-label="${label}"]`,
    });
  }
  // consider the other add workspace button
  // TODO UserSettings works as a drawer on desktop view after clicking on nav.headerDropdown.settings
  // TODO readonly userSettings= new UserSettings({ parent: this });
}

/**
 * Represents the HeaderDropdown in the NavigationSideBar component
 */
class HeaderDropdown extends DropdownMenu {
  readonly admin = this.menuItem('admin');
  readonly settings = this.menuItem('settings');
  readonly theme = this.menuItem('theme-toggle');
  readonly signOut = this.menuItem('sign-out');
}

/**
 * Represents the SidebarWorkspaceItem in the NavigationSideBar component
 */
class SidebarWorkspaceItem extends BaseComponent {
  readonly actionMenu = new WorkspaceActionDropdown({
    openMethod: async () => {
      await this.pwLocator.click({ button: 'right' });
    },
    root: this.root,
  });
}
