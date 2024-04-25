import { expect, Page } from '@playwright/test';

import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { safeName } from 'e2e/utils/naming';

export interface User {
  username: string;
  displayName: string | undefined;
  id: string;
  isAdmin: boolean;
  isActive: boolean;
}
interface UserArgs {
  username?: string;
  displayName?: string;
  isAdmin?: boolean;
  password?: string;
}

export class UserFixture {
  readonly userManagementPage: UserManagement;
  readonly #users = new Map<string, User>();

  constructor(readonly page: Page) {
    this.userManagementPage = new UserManagement(page);
  }

  async fillUserForm({ username, displayName, isAdmin, password }: UserArgs): Promise<void> {
    if (username !== undefined) {
      await this.userManagementPage.createUserModal.username.pwLocator.fill(username);
    }
    if (displayName !== undefined) {
      await this.userManagementPage.createUserModal.displayName.pwLocator.fill(displayName);
    }
    if (password !== undefined) {
      await this.userManagementPage.createUserModal.password.pwLocator.fill(password);
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

    await this.userManagementPage.createUserModal.footer.submit.pwLocator.click();
  }

  async createUser({
    username = safeName('test-user'),
    displayName,
    isAdmin,
    password = 'testPassword1',
  }: UserArgs = {}): Promise<User> {
    await this.userManagementPage.addUser.pwLocator.click();
    await expect(this.userManagementPage.createUserModal.pwLocator).toBeVisible();
    await expect(this.userManagementPage.createUserModal.header.title.pwLocator).toContainText(
      'Add User',
    );
    await this.fillUserForm({ displayName, isAdmin, password, username });
    await expect(this.userManagementPage.toast.pwLocator).toBeVisible();
    await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
      'New user has been created; advise user to change password as soon as possible.',
    );
    const row = await this.userManagementPage.getRowByUsernameSearch(username);
    const id = await row.getID();
    const user = { displayName, id, isActive: true, isAdmin: !!isAdmin, username };
    this.#users.set(String(id), user);
    return user;
  }

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
    await expect(this.userManagementPage.toast.pwLocator).toBeVisible();
    await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
      'User has been updated',
    );
    const editedUser = { ...user, ...edit };
    this.#users.set(String(user.id), editedUser);
    return editedUser;
  }

  async deactivateTestUsers(): Promise<void> {
    for await (const user of this.#users.values()) {
      if (user.isActive) {
        const actions = (await this.userManagementPage.getRowByUsernameSearch(user.username))
          .actions;
        await actions.pwLocator.click();
        if ((await actions.state.pwLocator.textContent()) !== 'Activate') {
          continue;
        }
        await actions.state.pwLocator.click();
        await expect(this.userManagementPage.toast.message.pwLocator).toContainText(
          'User has been deactivated',
        );
        this.#users.set(String(user.id), { ...user, isActive: false });
      }
    }
  }
}
