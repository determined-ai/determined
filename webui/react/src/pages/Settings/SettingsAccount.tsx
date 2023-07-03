import { Divider } from 'antd';
import React, { useCallback } from 'react';

import { Size } from 'components/Avatar';
import Button from 'components/kit/Button';
import InlineForm from 'components/kit/InlineForm';
import Input from 'components/kit/Input';
import { useModal } from 'components/kit/Modal';
import Avatar from 'components/kit/UserAvatar';
import PasswordChangeModalComponent from 'components/PasswordChangeModal';
import { patchUser } from 'services/api';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './SettingsAccount.module.scss';

export const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
export const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
export const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';
export const CHANGE_PASSWORD_TEXT = 'Change Password';

const SettingsAccount: React.FC = () => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const info = useObservable(determinedStore.info);

  const PasswordChangeModal = useModal(PasswordChangeModalComponent);

  const handleSaveDisplayName = useCallback(
    async (newValue: string | number): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: currentUser?.id || 0,
          userParams: { displayName: newValue as string },
        });
        userStore.updateUsers(user);
        message.success(API_DISPLAYNAME_SUCCESS_MESSAGE);
      } catch (e) {
        handleError(e, { silent: false, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser?.id],
  );

  const handleSaveUsername = useCallback(
    async (newValue: string | number): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: currentUser?.id || 0,
          userParams: { username: newValue as string },
        });
        userStore.updateUsers(user);
        message.success(API_USERNAME_SUCCESS_MESSAGE);
      } catch (e) {
        message.error(API_USERNAME_ERROR_MESSAGE);
        handleError(e, { silent: true, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser?.id],
  );

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip size={Size.ExtraLarge} user={currentUser} />
      </div>
      <Divider />
      <InlineForm
        inputValue={currentUser?.username}
        label="Username"
        required
        rules={[{ message: 'Please input your username', required: true }]}
        testId="username"
        onSubmit={handleSaveUsername}>
        <Input maxLength={32} placeholder="Add username" />
      </InlineForm>
      <Divider />
      <InlineForm
        inputValue={currentUser?.displayName}
        label="Display Name"
        testId="displayname"
        onSubmit={handleSaveDisplayName}>
        <Input maxLength={32} placeholder="Add display name" style={{ widows: '80%' }} />
      </InlineForm>
      {info.userManagementEnabled && (
        <>
          <Divider />
          <div className={css.row}>
            <label>Password</label>
            <Button onClick={PasswordChangeModal.open}>{CHANGE_PASSWORD_TEXT}</Button>
          </div>
          <PasswordChangeModal.Component />
        </>
      )}
    </div>
  );
};

export default SettingsAccount;
