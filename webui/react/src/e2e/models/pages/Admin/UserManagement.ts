import { AdminPage } from 'e2e/models/pages/Admin/index';
import { CreateUserModal } from 'e2e/models/components/CreateUserModal';
import { ChangeUserStatusModal } from 'e2e/models/components/ChangeUserStatusModal';
import { SetUserRolesModal } from 'e2e/models/components/SetUserRolesModal';
import { AddUsersToGroupsModal } from 'e2e/models/components/AddUsersToGroupsModal';
import { BaseComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the admin User Management page.
 * This constructor represents the contents in src/pages/Admin/UserManagement.tsx.
 * @param {Page} page - The '@playwright/test' Page being used by a test
 */
export class UserManagement extends AdminPage {
  static title: string = 'Determined';
  static url: string = 'admin/user-management';

  readonly actionRow: BaseComponent = new BaseComponent({ parent: this.content, selector: "data-testid='actionRow'" });

  readonly createUserModal: CreateUserModal = new CreateUserModal({ parent: this });
  readonly changeUserStatusModal: ChangeUserStatusModal = new ChangeUserStatusModal({ parent: this });
  readonly setUserRolesModal: SetUserRolesModal = new SetUserRolesModal({ parent: this });
  readonly addUsersToGroupsModal: AddUsersToGroupsModal = new AddUsersToGroupsModal({ parent: this });
}
