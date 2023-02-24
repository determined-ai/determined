import { Divider } from 'antd';
import React, { useCallback } from 'react';

import Button from 'components/kit/Button';
import Input from 'components/kit/Input';
import Avatar from 'components/kit/UserAvatar';
import useModalPasswordChange from 'hooks/useModal/UserSettings/useModalPasswordChange';
import { patchUser } from 'services/api';
import { Size } from 'shared/components/Avatar';
import { ErrorType } from 'shared/utils/error';
import { useCurrentUser, useUpdateUser } from 'stores/users';
import { message } from 'utils/dialogApi';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

import css from './SettingsAccount.module.scss';

export const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
export const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
export const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';
export const CHANGE_PASSWORD_TEXT = 'Change Password';

const SettingsAccount: React.FC = () => {
  const loadableCurrentUser = useCurrentUser();
  const updateUser = useUpdateUser();
  const currentUser = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });

  const { contextHolder: modalPasswordChangeContextHolder, modalOpen: openChangePasswordModal } =
    useModalPasswordChange();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [openChangePasswordModal]);

  const handleSaveDisplayName = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: currentUser?.id || 0,
          userParams: { displayName: newValue },
        });
        updateUser(user.id, (oldUser) => ({ ...oldUser, displayName: newValue }));
        message.success(API_DISPLAYNAME_SUCCESS_MESSAGE);
      } catch (e) {
        handleError(e, { silent: false, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser, updateUser],
  );

  const handleSaveUsername = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: currentUser?.id || 0,
          userParams: { username: newValue },
        });
        updateUser(user.id, (oldUser) => ({ ...oldUser, username: newValue }));
        message.success(API_USERNAME_SUCCESS_MESSAGE);
      } catch (e) {
        message.error(API_USERNAME_ERROR_MESSAGE);
        handleError(e, { silent: true, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser, updateUser],
  );

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip size={Size.ExtraLarge} user={currentUser} />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Username</label>
        <Input
          defaultValue={currentUser?.username ?? ''}
          maxLength={32}
          placeholder="Add username"
          onBlur={(e) => {
            const newValue = e.currentTarget.value;
            handleSaveUsername(newValue);
          }}
          onPressEnter={(e) => {
            e.currentTarget.blur();
          }}
        />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Display Name</label>
        <Input
          defaultValue={currentUser?.displayName ?? ''}
          maxLength={32}
          placeholder="Add display name"
          onBlur={(e) => {
            const newValue = e.currentTarget.value;
            handleSaveDisplayName(newValue);
          }}
          onPressEnter={(e) => {
            e.currentTarget.blur();
          }}
        />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Password</label>
        <Button onClick={handlePasswordClick}>{CHANGE_PASSWORD_TEXT}</Button>
      </div>
      {modalPasswordChangeContextHolder}
    </div>
  );
};

export default SettingsAccount;
