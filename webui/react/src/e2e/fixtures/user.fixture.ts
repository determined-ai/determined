import { Page } from '@playwright/test';
import _ from 'lodash';

import { expect } from 'e2e/fixtures/global-fixtures';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { safeName } from 'e2e/utils/naming';
import { repeatWithFallback } from 'e2e/utils/polling';
import { V1PostUserRequest } from 'services/api-ts-sdk/api';

interface UserArgs {
  username?: string;
  displayName?: string;
  admin?: boolean;
  password?: string;
}

export class UserFixture {
  readonly userManagementPage: UserManagement;

  constructor(readonly page: Page) {
    this.userManagementPage = new UserManagement(page);
  }

  /**
   * Fills the create/edit user form and submits it.
   * @param {V1PostUserRequest} edit user settings to set user too.
   */
  async fillUserForm(edit: UserArgs): Promise<void> {
    await this.userManagementPage._page.waitForTimeout(500); // ant/Popover - menus may reset input shortly after opening [ET-283]
    if (edit.username !== undefined) {
      await this.userManagementPage.createUserModal.username.pwLocator.fill(edit.username);
    }
    if (edit.displayName !== undefined) {
      await this.userManagementPage.createUserModal.displayName.pwLocator.fill(edit.displayName);
    }
    if (edit.password !== undefined) {
      await this.userManagementPage.createUserModal.password.pwLocator.fill(edit.password);
      await this.userManagementPage.createUserModal.confirmPassword.pwLocator.fill(edit.password);
    }

    const checkedAttribute =
      await this.userManagementPage.createUserModal.adminToggle.pwLocator.getAttribute(
        'aria-checked',
      );
    if (checkedAttribute === null) {
      throw new Error('Expected attribute aria-checked to be present.');
    }
    const adminState = JSON.parse(checkedAttribute);
    if (!!edit.admin !== adminState) {
      await this.userManagementPage.createUserModal.adminToggle.pwLocator.click();
    }

    // password and username are required to create a user; if these are filled, submit should be enabled
    await expect(
      this.userManagementPage.createUserModal.footer.submit.pwLocator,
    ).not.toBeDisabled();
    await this.userManagementPage.createUserModal.footer.submit.pwLocator.click();
  }

  /**
   * Creates a user with the given parameters via the UI.
   * @param {UserArgs} obj
   * @param {string} [obj.username] - The username to create
   * @param {string} [obj.displayName] - The display name to create
   * @param {boolean} [obj.isAdmin] - Whether the user should be an admin
   * @param {string} [obj.password] - Password to set
   * @returns {Promise<User>} Representation of the created user
   */
  async createUser({
    username = 'test-user',
    displayName,
    admin = false,
    password = 'TestPassword1',
  }: UserArgs = {}): Promise<V1PostUserRequest> {
    const safeUsername = safeName(username);
    await expect(
      repeatWithFallback(
        async () => {
          await this.userManagementPage.addUser.pwLocator.click();
        },
        async () => {
          // unfortunately, this can fail on CI sometimes. this is to deflake
          await this.userManagementPage.goto();
        },
      ),
    ).toPass({ timeout: 15_000 });
    await expect(this.userManagementPage.createUserModal.pwLocator).toBeVisible();
    await expect(this.userManagementPage.createUserModal.header.title.pwLocator).toContainText(
      'Add User',
    );

    await this.fillUserForm({ admin, displayName, password, username: safeUsername });
    // Hashing a password after form submit might take a little extra time, so this can be a slower operation
    await expect(this.userManagementPage.toast.pwLocator).toBeVisible({ timeout: 10_000 });
    await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
      'New user has been created; advise user to change password as soon as possible.',
    );
    await this.userManagementPage.toast.close.pwLocator.click();
    await expect(this.userManagementPage.toast.pwLocator).toHaveCount(0);
    const row = await this.userManagementPage.getRowByUsernameSearch(safeUsername);
    const id = parseInt(await row.getId());
    const user: V1PostUserRequest = {
      isHashed: false,
      password,
      user: {
        active: true,
        admin: !!admin,
        displayName,
        id,
        username: safeUsername,
      },
    };
    return user;
  }

  /**
   * Edit a user with the given parameters via the UI.
   * @param {User} user - Representation of the user to edit
   * @param {UserArgs} edit
   * @param {string} [edit.username] - The username to edit
   * @param {string} [edit.displayName] - The display name to edit
   * @param {boolean} [edit.isAdmin] - Whether the user should be an admin
   * @returns {Promise<User>} Representation of the edited user
   */
  async editUser(user: V1PostUserRequest, edit: UserArgs = {}): Promise<V1PostUserRequest> {
    if (user.user === undefined) {
      throw new Error('Trying to edit user that is Undefined.');
    }
    const editedUser = { ...user };
    await this.singleUserSearchAndEdit(user);
    await expect(this.userManagementPage.createUserModal.username.pwLocator).toBeDisabled();
    expect(
      await this.userManagementPage.createUserModal.username.pwLocator.getAttribute('value'),
    ).toEqual(user.user.username || '');
    await expect(
      repeatWithFallback(
        async () => await this.fillUserForm(edit),
        async () => await this.singleUserSearchAndEdit(user),
      ),
    ).toPass({ timeout: 30_000 });
    await expect(this.userManagementPage.toast.pwLocator).toBeVisible({ timeout: 10_000 }); // this can be slow if the backend write is slow
    await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
      'User has been updated',
    );
    await this.userManagementPage.toast.close.pwLocator.click();
    await expect(this.userManagementPage.toast.pwLocator).toHaveCount(0);
    if (editedUser.user === undefined) {
      throw new Error('Result from edit user is Undefined.');
    }
    if (edit.password !== undefined) editedUser.password = edit.password;
    if (edit.admin !== undefined) editedUser.user.admin = edit.admin;
    if (edit.displayName !== undefined) editedUser.user.displayName = edit.displayName;
    return editedUser;
  }

  private async singleUserSearchAndEdit(user: V1PostUserRequest) {
    await expect(
      repeatWithFallback(
        async () => {
          const row = await this.userManagementPage.getRowByUsernameSearch(
            user.user?.username ?? '',
          );
          await (await row.actions.open()).edit.pwLocator.click();
          await expect(this.userManagementPage.createUserModal.pwLocator).toBeVisible();
          await expect(
            this.userManagementPage.createUserModal.header.title.pwLocator,
          ).toContainText('Edit User');
        },
        async () => {
          await this.userManagementPage.goto(); // If the table refreshes right on the 'Edit User' click it can close the modal
        },
      ),
    ).toPass({ timeout: 15_000 });
  }

  /**
   * Validate a user via the UI matches the expected.
   * @param {User} obj - Representation of the user to validate against the table
   */
  async validateUser({ user }: V1PostUserRequest): Promise<void> {
    if (user === undefined) {
      throw new Error('Can not validate undefined user.');
    }
    const row = await this.userManagementPage.getRowByUsernameSearch(user.username);
    expect(Number(await row.getId())).toEqual(user.id);
    await expect(row.user.name.pwLocator).toContainText(user.username);
    if (user.displayName) {
      await expect(row.user.alias.pwLocator).toContainText(user.displayName, { timeout: 10_000 });
    } else {
      await row.user.alias.pwLocator.waitFor({ state: 'hidden' });
    }
    await expect(row.role.pwLocator).toContainText(user.admin ? 'Admin' : 'Member');
    await expect(row.status.pwLocator).toContainText(user.active ? 'Active' : 'Inactive');
  }

  /**
   * Deactivates all users present on the table.
   */
  async deactivateTestUsersOnTable(): Promise<void> {
    // select all users
    await this.userManagementPage.table.table.headRow.selectAll.pwLocator.click();
    await expect(this.userManagementPage.table.table.headRow.selectAll.pwLocator).toBeChecked();
    // open group actions
    await (await this.userManagementPage.actions.open()).status.pwLocator.click();
    // deactivate
    await this.userManagementPage.changeUserStatusModal.pwLocator.waitFor();
    await this.userManagementPage.changeUserStatusModal.status.openMenu();
    await this.userManagementPage.changeUserStatusModal.status.deactivate.pwLocator.click();
    await this.userManagementPage.changeUserStatusModal.footer.submit.pwLocator.click();
  }

  /**
   * Changes the status of a user.
   * @param {User} user - The user to change the status of
   * @param {boolean} activate - Whether to activate or deactivate the user
   * @returns {Promise<User>} The updated user
   */
  async changeStatusUser(user: V1PostUserRequest, activate: boolean): Promise<V1PostUserRequest> {
    if (user.user?.active === activate) {
      return user;
    }
    await expect(async () => {
      if (user.user === undefined) {
        throw new Error('Trying to change status on user that is Undefined.');
      }
      // user table can flake if running in parrallel
      const actions = (await this.userManagementPage.getRowByUsernameSearch(user.user.username))
        .actions;
      await actions.open();
      if (
        (await actions.state.pwLocator.textContent()) !== (activate ? 'Activate' : 'Deactivate')
      ) {
        return;
      }
      await actions.state.pwLocator.click();
      await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
        activate ? 'User has been activated' : 'User has been deactivated',
      );
      await this.userManagementPage.toast.close.pwLocator.click();
    }).toPass({ timeout: 35_000 });
    const editedUser = _.merge(user, { user: { active: activate } });
    return editedUser;
  }
}
