import { expect, Page } from '@playwright/test';

import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { safeName } from 'e2e/utils/naming';
import { repeatWithFallback } from 'e2e/utils/polling';

export interface User {
  username: string;
  displayName: string | undefined;
  id: string;
  isAdmin: boolean;
  isActive: boolean;
  password: string;
}
interface UserArgs {
  username?: string;
  displayName?: string;
  isAdmin?: boolean;
  password?: string;
}

// One list of users per test session. This is to encourage a final teardown
// instance of the user fixture to deactivate all users created by the different
// instances of the fixture used in each test scenario.
// Note: This is can't collide when running tests in parallel because playwright
// workers can't share variables.
const users = new Map<string, User>();

export class UserFixture {
  readonly userManagementPage: UserManagement;

  constructor(readonly page: Page) {
    this.userManagementPage = new UserManagement(page);
  }

  /**
   * Fills the create/edit user form and submits it.
   * @param {UserArgs} obj
   * @param {string} [obj.username] - The username to fill in the form
   * @param {string} [obj.displayName] - The display name to fill in the form
   * @param {boolean} [obj.isAdmin] - Whether the user should be an admin
   * @param {string} [obj.password] - The password to fill in the form
   */
  async fillUserForm({ username, displayName, isAdmin, password }: UserArgs): Promise<void> {
    if (username !== undefined) {
      await this.userManagementPage.createUserModal.username.pwLocator.fill(username);
    }
    if (displayName !== undefined) {
      await this.userManagementPage.createUserModal.displayName.pwLocator.fill(displayName);
    }
    if (password !== undefined) {
      await this.userManagementPage.createUserModal.password.pwLocator.fill(password);
      await this.userManagementPage.createUserModal.confirmPassword.pwLocator.fill(password);
    }

    const checkedAttribute =
      await this.userManagementPage.createUserModal.adminToggle.pwLocator.getAttribute(
        'aria-checked',
      );
    if (checkedAttribute === null) {
      throw new Error('Expected attribute aria-checked to be present.');
    }
    const adminState = JSON.parse(checkedAttribute);
    if (!!isAdmin !== adminState) {
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
    isAdmin,
    password = 'TestPassword1',
  }: UserArgs = {}): Promise<User> {
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
    ).toPass({ timeout: 15000 });
    await expect(this.userManagementPage.createUserModal.pwLocator).toBeVisible();
    await expect(this.userManagementPage.createUserModal.header.title.pwLocator).toContainText(
      'Add User',
    );
    await this.fillUserForm({ displayName, isAdmin, password, username: safeUsername });
    // Hashing a password after form submit might take a little extra time, so this can be a slower operation
    await expect(this.userManagementPage.toast.pwLocator).toBeVisible({ timeout: 10_000 });
    await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
      'New user has been created; advise user to change password as soon as possible.',
    );
    await this.userManagementPage.toast.close.pwLocator.click();
    await expect(this.userManagementPage.toast.pwLocator).toHaveCount(0);
    const row = await this.userManagementPage.getRowByUsernameSearch(safeUsername);
    const id = await row.getId();
    const user = {
      displayName,
      id,
      isActive: true,
      isAdmin: !!isAdmin,
      password,
      username: safeUsername,
    };
    users.set(String(id), user);
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
  async editUser(user: User, edit: UserArgs = {}): Promise<User> {
    const row = await this.userManagementPage.getRowByUsernameSearch(user.username);
    await row.actions.pwLocator.click();
    await row.actions.edit.pwLocator.click();
    await expect(this.userManagementPage.createUserModal.pwLocator).toBeVisible();
    await expect(this.userManagementPage.createUserModal.header.title.pwLocator).toContainText(
      'Edit User',
    );
    await expect(this.userManagementPage.createUserModal.username.pwLocator).toBeDisabled();
    expect(
      await this.userManagementPage.createUserModal.displayName.pwLocator.getAttribute('value'),
    ).toEqual(user.displayName || '');
    const checkedAttribute =
      await this.userManagementPage.createUserModal.adminToggle.pwLocator.getAttribute(
        'aria-checked',
      );
    if (checkedAttribute === null) {
      throw new Error('Expected attribute aria-checked to be present.');
    }
    const adminState = JSON.parse(checkedAttribute);
    if (user.isAdmin) {
      expect(adminState).toBeTruthy();
    } else {
      expect(adminState).not.toBeTruthy();
    }
    await this.fillUserForm(edit);
    await expect(this.userManagementPage.toast.pwLocator).toBeVisible({ timeout: 5_000 });
    await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
      'User has been updated',
    );
    await this.userManagementPage.toast.close.pwLocator.click();
    await expect(this.userManagementPage.toast.pwLocator).toHaveCount(0);
    const editedUser = { ...user, ...edit };
    users.set(String(user.id), editedUser);
    return editedUser;
  }

  /**
   * Delete a user via the UI.
   * @param {User} obj - Representation of the user to validate against the table
   */
  async validateUser({ username, displayName, id, isAdmin, isActive }: User): Promise<void> {
    const row = await this.userManagementPage.getRowByUsernameSearch(username);
    expect(await row.getId()).toEqual(id);
    await expect(row.user.name.pwLocator).toContainText(username);
    if (displayName) {
      await expect(row.user.alias.pwLocator).toContainText(displayName);
    } else {
      await row.user.alias.pwLocator.waitFor({ state: 'hidden' });
    }
    await expect(row.role.pwLocator).toContainText(isAdmin ? 'Admin' : 'Member');
    await expect(row.status.pwLocator).toContainText(isActive ? 'Active' : 'Inactive');
  }

  /**
   * Deactivates all users present on the table.
   */
  async deactivateTestUsersOnTable(): Promise<void> {
    // get all user ids so we can update the status later
    const ids = await this.userManagementPage.table.table.allRowKeys();
    // select all users
    await this.userManagementPage.actions.pwLocator.waitFor({ state: 'hidden' });
    await this.userManagementPage.table.table.headRow.selectAll.pwLocator.click();
    await expect(this.userManagementPage.table.table.headRow.selectAll.pwLocator).toBeChecked();
    // open group actions
    await this.userManagementPage.actions.pwLocator.click();
    await this.userManagementPage.actions.status.pwLocator.click();
    // deactivate
    await this.userManagementPage.changeUserStatusModal.pwLocator.waitFor();
    await this.userManagementPage.changeUserStatusModal.status.pwLocator.click();
    await this.userManagementPage.changeUserStatusModal.status.deactivate.pwLocator.click();
    await this.userManagementPage.changeUserStatusModal.footer.submit.pwLocator.click();
    for (const id of ids) {
      const user = users.get(id);
      if (user === undefined) {
        throw new Error(
          `Expected user with id ${id} present on the table to have been created during this session`,
        );
      }
      users.set(String(id), { ...user, isActive: false });
    }
  }

  /**
   * Changes the status of a user.
   * @param {User} user - The user to change the status of
   * @param {boolean} activate - Whether to activate or deactivate the user
   * @returns {Promise<User>} The updated user
   */
  async changeStatusUser(user: User, activate: boolean): Promise<User> {
    if (user.isActive === activate) {
      return user;
    }
    await expect(async () => {
      // user table can flake if running in parrallel
      const actions = (await this.userManagementPage.getRowByUsernameSearch(user.username)).actions;
      await actions.pwLocator.click();
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
    }).toPass({ timeout: 15000 });
    const editedUser = { ...user, isActive: activate };
    users.set(String(user.id), editedUser);
    return editedUser;
  }

  /**
   * Deactivates all test users created during this session.
   */
  async deactivateAllTestUsers(): Promise<void> {
    for await (const user of users.values()) {
      await this.changeStatusUser(user, false);
    }
  }
}
