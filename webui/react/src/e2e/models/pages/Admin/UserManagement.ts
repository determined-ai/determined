import { expect, Page } from '@playwright/test';

import { BaseComponent } from 'e2e/models/BaseComponent';
import { AddUsersToGroupsModal } from 'e2e/models/components/AddUsersToGroupsModal';
import { ChangeUserStatusModal } from 'e2e/models/components/ChangeUserStatusModal';
import { CreateUserModal } from 'e2e/models/components/CreateUserModal';
import { SetUserRolesModal } from 'e2e/models/components/SetUserRolesModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';
import { Dropdown } from 'e2e/models/hew/Dropdown';
import { Toast } from 'e2e/models/hew/Toast';
import { AdminPage } from 'e2e/models/pages/Admin/index';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class UserManagement extends AdminPage {
  readonly title: RegExp = /(Determined|HPE Machine Learning Development Environment)/;
  readonly url: string = 'admin/user-management';
  readonly getRowByID: (value: string) => UserRow;

  readonly #actionRow: BaseComponent = new BaseComponent({
    parent: this.pivot.tabContent,
    selector: '[data-testid="actionRow"]',
  });
  readonly search: BaseComponent = new BaseComponent({
    parent: this.#actionRow,
    selector: '[data-testid="search"]',
  });
  // TODO do these selects work?
  readonly filterRole: BaseComponent = new BaseComponent({
    parent: this.#actionRow,
    selector: '[data-testid="roleSelect"]',
  });
  // TODO do these selects work?
  readonly filterStatus: BaseComponent = new BaseComponent({
    parent: this.#actionRow,
    selector: '[data-testid="statusSelect"]',
  });
  readonly addUser: BaseComponent = new BaseComponent({
    parent: this.#actionRow,
    selector: '[data-testid="addUser"]',
  });

  readonly table: InteractiveTable<UserRow, UserHeadRow> = new InteractiveTable({
    headRowType: UserHeadRow,
    parent: this.pivot.tabContent,
    rowType: UserRow,
  });
  readonly skeletonTable: SkeletonTable = new SkeletonTable({ parent: this.pivot.tabContent });

  readonly createUserModal: CreateUserModal = new CreateUserModal({ parent: this });
  readonly changeUserStatusModal: ChangeUserStatusModal = new ChangeUserStatusModal({
    parent: this,
  });
  readonly setUserRolesModal: SetUserRolesModal = new SetUserRolesModal({ parent: this });
  readonly addUsersToGroupsModal: AddUsersToGroupsModal = new AddUsersToGroupsModal({
    parent: this,
  });
  readonly toast = new Toast({
    attachment: Toast.selectorTopRight,
    parent: this,
  });

  constructor(page: Page) {
    super(page);
    this.getRowByID = this.table.table.getRowByDataKey;
  }

  /**
   * Returns a row that matches a given username
   * @param {string} name - The username to filter UserTable rows by
   */
  async getRowByUsername(name: string): Promise<UserRow> {
    const filteredRows = await this.table.table.filterRows(async (row: UserRow) => {
      return (await row.user.pwLocator.innerText()).includes(name);
    });
    expect(
      filteredRows,
      `name:${name}, component:${await this.table.table.rows.pwLocator.allInnerTexts()}`,
    ).toHaveLength(1);
    return filteredRows[0];
  }

  /**
   * Returns a row that matches a given username
   * @param {string} name - The username to filter UserTable rows by
   */
  async getRowByUsernameSearch(name: string): Promise<UserRow> {
    await this.search.pwLocator.clear();
    await Promise.all([
      this.search.pwLocator.fill(name),
      expect(this.table.table.rows.pwLocator).toHaveCount(1),
    ]);
    const filteredRows = await this.table.table.filterRows(async (row: UserRow) => {
      return (await row.user.pwLocator.innerText()).includes(name);
    });
    expect(
      filteredRows,
      `name:${name}, component:${await this.table.table.rows.pwLocator.allInnerTexts()}`,
    ).toHaveLength(1);
    return filteredRows[0];
  }
}

/**
 * Returns the representation of the InteractiveTable header row defined by the User Admin page.
 * This constructor represents the InteractiveTable in src/pages/Admin/UserManagement.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this UserHeadRow
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class UserHeadRow extends HeadRow {
  readonly user: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="User"]',
  });
  readonly status: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="Status"]',
  });
  readonly lastSeen: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="Last Seen"]',
  });
  readonly role: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="Role"]',
  });
  readonly remote: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="Remote"]',
  });
  readonly modified: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="Modified"]',
  });
}

/**
 * Returns the representation of the InteractiveTable row defined by the User Admin page.
 * This constructor represents the InteractiveTable in src/pages/Admin/UserManagement.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this UserRow
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class UserRow extends Row {
  // If you're wondering where (1) is, it's the checkbox column (smelly)
  // TODO consider nameplate component
  readonly user: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="user"]',
  });
  readonly status: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="status"]',
  });
  readonly lastSeen: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="lastSeen"]',
  });
  readonly role: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="role"]',
  });
  readonly remote: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="remote"]',
  });
  readonly modified: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="modified"]',
  });
  readonly actions: UserActionDropdown = new UserActionDropdown({
    parent: this,
    selector: '[data-testid="actions"]',
  });
}

/**
 * Returns the representation of the ActionDropdown menu defined by the User Admin page.
 * This constructor represents the InteractiveTable in src/pages/Admin/UserManagement.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this UserActionDropdown
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
class UserActionDropdown extends Dropdown {
  readonly edit: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('edit'),
  });
  readonly agent: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('agent'),
  });
  readonly state: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('state'),
  });
}
