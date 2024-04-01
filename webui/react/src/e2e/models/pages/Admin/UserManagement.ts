import { Page } from '@playwright/test';
import { BaseComponent } from 'e2e/models/BaseComponent';
import { AddUsersToGroupsModal } from 'e2e/models/components/AddUsersToGroupsModal';
import { ChangeUserStatusModal } from 'e2e/models/components/ChangeUserStatusModal';
import { CreateUserModal } from 'e2e/models/components/CreateUserModal';
import { SetUserRolesModal } from 'e2e/models/components/SetUserRolesModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';
import { Dropdown } from 'e2e/models/hew/Dropdown';
import { AdminPage } from 'e2e/models/pages/Admin/index';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class UserManagement extends AdminPage {
  static title: string = 'Determined';
  static url: string = 'admin/user-management';
  readonly getRowByID: (value: string) => UserRow;

  readonly actionRow: BaseComponent = new BaseComponent({
    parent: this.content,
    selector: 'data-testid="actionRow"',
  });
  readonly table: InteractiveTable<UserRow, UserHeadRow> = new InteractiveTable({
    headRowType: UserHeadRow,
    parent: this.content,
    rowType: UserRow,
  });
  readonly skeletonTable: SkeletonTable = new SkeletonTable({ parent: this.content });

  readonly createUserModal: CreateUserModal = new CreateUserModal({ parent: this });
  readonly changeUserStatusModal: ChangeUserStatusModal = new ChangeUserStatusModal({
    parent: this,
  });
  readonly setUserRolesModal: SetUserRolesModal = new SetUserRolesModal({ parent: this });
  readonly addUsersToGroupsModal: AddUsersToGroupsModal = new AddUsersToGroupsModal({
    parent: this,
  });

  constructor(page: Page) {
    super(page)
    this.getRowByID = this.table.table.getRowByDataKey;
  }
}

class UserHeadRow extends HeadRow {
  readonly user: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'data-testid="User"',
  });
  readonly status: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'data-testid="Status"',
  });
  readonly lastSeen: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'data-testid="Last Seen"',
  });
  readonly role: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'data-testid="Role"',
  });
  readonly remote: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'data-testid="Remote"',
  });
  readonly modified: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'data-testid="Modified"',
  });
}
class UserRow extends Row {
  /**
   * Returns a templated selector for children components.
   * TODO DANGER - dependant on order or elements. This is a smelly practice
   * Consider grabbing column number from header, or including data-testid
   * @param {string} n - cell number
   */
  static selectorTemplate(n: number): string {
    return `td.ant-table-cell:nth-of-type(${n})`;
  }
  // If you're wondering where (1) is, it's the checkbox column (smelly)
  // TODO consider nameplate component
  readonly user: BaseComponent = new BaseComponent({
    parent: this,
    selector: UserRow.selectorTemplate(2),
  });
  readonly status: BaseComponent = new BaseComponent({
    parent: this,
    selector: UserRow.selectorTemplate(3),
  });
  readonly lastSeen: BaseComponent = new BaseComponent({
    parent: this,
    selector: UserRow.selectorTemplate(4),
  });
  readonly role: BaseComponent = new BaseComponent({
    parent: this,
    selector: UserRow.selectorTemplate(5),
  });
  readonly remote: BaseComponent = new BaseComponent({
    parent: this,
    // this is intentional and smelly
    selector: UserRow.selectorTemplate(5),
  });
  readonly modified: BaseComponent = new BaseComponent({
    parent: this,
    selector: UserRow.selectorTemplate(6),
  });
  readonly actions: UserActionDropdown = new UserActionDropdown({
    parent: this,
    selector: UserRow.selectorTemplate(7),
  });
}

class UserActionDropdown extends Dropdown {
  readonly admin: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('edit'),
  });
  readonly settings: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('agent'),
  });
  readonly theme: BaseComponent = new BaseComponent({
    parent: this.menu,
    selector: Dropdown.selectorTemplate('state'),
  });
}
