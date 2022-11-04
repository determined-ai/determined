import { Button, Divider, message } from 'antd';
import React, { useCallback } from 'react';

import InlineEditor from 'components/InlineEditor';
import Avatar from 'components/UserAvatar';
import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import useModalPasswordChange from 'hooks/useModal/UserSettings/useModalPasswordChange';
import { patchUser } from 'services/api';
import { Size } from 'shared/components/Avatar';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import css from './SettingsAccount.module.scss';

export const API_DISPLAYNAME_ERROR_MESSAGE = 'Could not update display name.';
export const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
export const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
export const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';
export const CHANGE_PASSWORD_TEXT = 'Change Password';

const SettingsAccount: React.FC = () => {
  const { auth } = useStore();
  const storeDispatch = useStoreDispatch();

  const { contextHolder: modalPasswordChangeContextHolder, modalOpen: openChangePasswordModal } =
    useModalPasswordChange();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [openChangePasswordModal]);

  const handleSaveDisplayName = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: auth.user?.id || 0,
          userParams: { displayName: newValue },
        });
        storeDispatch({ type: StoreAction.SetCurrentUser, value: user });
        message.success(API_DISPLAYNAME_SUCCESS_MESSAGE);
      } catch (e) {
        message.error(API_DISPLAYNAME_ERROR_MESSAGE);
        handleError(e, { silent: true, type: ErrorType.Input });
        return e as Error;
      }
    },
    [auth.user, storeDispatch],
  );

  const handleSaveUsername = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        const user = await patchUser({
          userId: auth.user?.id || 0,
          userParams: { username: newValue },
        });
        storeDispatch({ type: StoreAction.SetCurrentUser, value: user });
        message.success(API_USERNAME_SUCCESS_MESSAGE);
      } catch (e) {
        message.error(API_USERNAME_ERROR_MESSAGE);
        handleError(e, { silent: true, type: ErrorType.Input });
        return e as Error;
      }
    },
    [auth.user, storeDispatch],
  );

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip size={Size.ExtraLarge} userId={auth.user?.id} />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Username</label>
        <InlineEditor
          maxLength={32}
          pattern={new RegExp('^[a-z][a-z0-9]*$', 'i')}
          placeholder="Add username"
          value={auth.user?.username || ''}
          onSave={handleSaveUsername}
        />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Display Name</label>
        <InlineEditor
          maxLength={32}
          pattern={new RegExp('^[a-z][a-z0-9\\s]*$', 'i')}
          placeholder="Add display name"
          value={auth.user?.displayName || ''}
          onSave={handleSaveDisplayName}
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
