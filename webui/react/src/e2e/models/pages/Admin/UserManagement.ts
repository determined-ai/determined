import { BaseComponent } from 'e2e/models/BaseComponent';
import { AddUsersToGroupsModal } from 'e2e/models/components/AddUsersToGroupsModal';
import { ChangeUserStatusModal } from 'e2e/models/components/ChangeUserStatusModal';
import { CreateUserModal } from 'e2e/models/components/CreateUserModal';
import { SetUserRolesModal } from 'e2e/models/components/SetUserRolesModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';
import { SkeletonTable } from 'e2e/models/components/Table/SkeletonTable';
import { AdminPage } from 'e2e/models/pages/Admin/index';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class UserManagement extends AdminPage {
  static title: string = 'Determined';
  static url: string = 'admin/user-management';

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
}

class UserHeadRow extends HeadRow {
  readonly user: BaseComponent = new BaseComponent({ parent: this, selector: `data-testid="User"` });
  readonly status: BaseComponent = new BaseComponent({ parent: this, selector: `data-testid="Status"` });
  readonly lastSeen: BaseComponent = new BaseComponent({ parent: this, selector: `data-testid="Last Seen"` });
  readonly role: BaseComponent = new BaseComponent({ parent: this, selector: `data-testid="Role"` });
  readonly modified: BaseComponent = new BaseComponent({ parent: this, selector: `data-testid="Modified"` });
}
class UserRow extends Row {}
