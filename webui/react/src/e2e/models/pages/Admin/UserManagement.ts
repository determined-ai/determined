import { Page } from '@playwright/test';

import { expect } from 'e2e/fixtures/global-fixtures';
import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { Select } from 'e2e/models/common/hew/Select';
import { Toast } from 'e2e/models/common/hew/Toast';
import { AddUsersToGroupsModal } from 'e2e/models/components/AddUsersToGroupsModal';
import { ChangeUserStatusModal } from 'e2e/models/components/ChangeUserStatusModal';
import { CreateUserModal } from 'e2e/models/components/CreateUserModal';
import { SetUserRolesModal } from 'e2e/models/components/SetUserRolesModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';
import { UserBadge } from 'e2e/models/components/UserBadge';
import { AdminPage } from 'e2e/models/pages/Admin/index';

/**
 * Represents the UserManagement page from src/pages/Admin/UserManagement.tsx
 */
export class UserManagement extends AdminPage {
  readonly url = 'admin/user-management';
  readonly getRowById: (value: string) => UserRow;

  readonly #actionRow = new BaseComponent({
    parent: this.pivot.tabContent,
    selector: '[data-testid="actionRow"]',
  });
  readonly search = new BaseComponent({
    parent: this.#actionRow,
    selector: '[data-testid="search"]',
  });
  readonly filterRole = new RoleSelect({
    parent: this.#actionRow,
    selector: '[data-testid="roleSelect"]',
  });
  readonly filterStatus = new StatusSelect({
    parent: this.#actionRow,
    selector: '[data-testid="statusSelect"]',
  });
  readonly actions = new ActionDropdownMenu({
    clickThisComponentToOpen: new BaseComponent({
      parent: this.#actionRow,
      selector: '[data-testid="actions"]',
    }),
    root: this,
  });
  readonly addUser = new BaseComponent({
    parent: this.#actionRow,
    selector: '[data-testid="addUser"]',
  });

  readonly table = new InteractiveTable({
    parent: this.pivot.tabContent,
    tableArgs: {
      attachment: '[data-testid="table"]',
      headRowType: UserHeadRow,
      rowType: UserRow,
    },
  });
  readonly skeletonTable = new SkeletonTable({ parent: this.pivot.tabContent });

  readonly createUserModal = new CreateUserModal({ root: this });
  readonly changeUserStatusModal = new ChangeUserStatusModal({
    root: this,
  });
  readonly setUserRolesModal = new SetUserRolesModal({ root: this });
  readonly addUsersToGroupsModal = new AddUsersToGroupsModal({
    root: this,
  });
  readonly toast = new Toast({
    attachment: Toast.selectorTopRight,
    parent: this,
  });

  constructor(page: Page) {
    super(page);
    this.getRowById = this.table.table.getRowByDataKey;
  }

  /**
   * Returns a row that matches a given username
   * @param {string} name - The username to filter UserTable rows by
   */
  async getRowByUsername(name: string): Promise<UserRow> {
    const filteredRows = await this.table.table.filterRows(async (row: UserRow) => {
      return (await row.user.name.pwLocator.innerText()).includes(name);
    });

    const removeNewLines = (item: string) => item.replace(/(\r\n|\n|\r)/gm, '');
    expect(
      filteredRows,
      `name: ${name}
      users: ${(await this.table.table.rows.user.pwLocator.allInnerTexts()).map(removeNewLines)}
      table: ${(await this.table.table.rows.pwLocator.allInnerTexts()).map(removeNewLines)}`,
    ).toHaveLength(1);
    return filteredRows[0];
  }

  /**
   * Searches for a user and returns a row that matches
   * @param {string} name - The username to filter UserTable rows by
   */
  async getRowByUsernameSearch(name: string): Promise<UserRow> {
    await expect(async () => {
      // user table can flake if running in parrallel
      await this.search.pwLocator.clear();
      await expect(this.table.table.rows.pwLocator).not.toHaveCount(1);
      await this.search.pwLocator.fill(name);
      await expect(this.table.table.rows.pwLocator).toHaveCount(1);
    }).toPass({ timeout: 15_000 });
    return await this.getRowByUsername(name);
  }
}

/**
 * Represents the head row from the table in src/pages/Admin/UserManagement.tsx
 */
class UserHeadRow extends HeadRow {
  readonly user = new BaseComponent({
    parent: this,
    selector: '[data-testid="User"]',
  });
  readonly status = new BaseComponent({
    parent: this,
    selector: '[data-testid="Status"]',
  });
  readonly lastSeen = new BaseComponent({
    parent: this,
    selector: '[data-testid="Last Seen"]',
  });
  readonly role = new BaseComponent({
    parent: this,
    selector: '[data-testid="Role"]',
  });
  readonly remote = new BaseComponent({
    parent: this,
    selector: '[data-testid="Remote"]',
  });
  readonly modified = new BaseComponent({
    parent: this,
    selector: '[data-testid="Modified"]',
  });
}

/**
 * Represents a row from the table in src/pages/Admin/UserManagement.tsx
 */
class UserRow extends Row {
  readonly user = new UserBadge({
    parent: this,
    selector: '[data-testid="user"]',
  });
  readonly status = new BaseComponent({
    parent: this,
    selector: '[data-testid="status"]',
  });
  readonly lastSeen = new BaseComponent({
    parent: this,
    selector: '[data-testid="lastSeen"]',
  });
  readonly role = new BaseComponent({
    parent: this,
    selector: '[data-testid="role"]',
  });
  readonly remote = new BaseComponent({
    parent: this,
    selector: '[data-testid="remote"]',
  });
  readonly modified = new BaseComponent({
    parent: this,
    selector: '[data-testid="modified"]',
  });
  readonly actions = new UserActionDropdown({
    clickThisComponentToOpen: new BaseComponent({
      parent: this,
      selector: '[data-testid="actions"]',
    }),
    root: this.root,
  });
}

/**
 * Represents the UserActionDropdown from src/pages/Admin/UserManagement.tsx
 */
class UserActionDropdown extends DropdownMenu {
  readonly edit = this.menuItem('edit');
  readonly agent = this.menuItem('agent');
  readonly state = this.menuItem('state');
}

/**
 * Represents the ActionDropdownMenu from src/pages/Admin/UserManagement.tsx
 */
class ActionDropdownMenu extends DropdownMenu {
  readonly status = this.menuItem('change-status');
  readonly roles = this.menuItem('set-roles');
  readonly groups = this.menuItem('add-to-groups');
}

/**
 * Represents the role Select from src/pages/Admin/UserManagement.tsx
 */
class RoleSelect extends Select {
  readonly allRoles = this.menuItem('All Roles');
  readonly admin = this.menuItem('Admin');
  readonly nonAdmin = this.menuItem('Non-Admin');
}

/**
 * Represents the status Select from src/pages/Admin/UserManagement.tsx
 */
class StatusSelect extends Select {
  readonly allStatuses = this.menuItem('All Statuses');
  readonly activeUsers = this.menuItem('Active Users');
  readonly deactivatedUsers = this.menuItem('Deactivated Users');
}
