import { Button, Divider, message } from 'antd';
import React, { useCallback } from 'react';

import InlineEditor from 'components/InlineEditor';
import Avatar from 'components/UserAvatar';
import { CharLength } from 'constants/values';
import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import useModalPasswordChange from 'hooks/useModal/UserSettings/useModalPasswordChange';
import { patchUser } from 'services/api';
import { Size } from 'shared/components/Avatar';
import { ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import css from './SettingsAccount.module.scss';

export const CHANGE_PASSWORD_TEXT = 'Change Password';
export const API_SUCCESS_MESSAGE = 'Display name updated.';
export const API_ERROR_MESSAGE = 'Could not update display name.';

const SettingsAccount: React.FC = () => {
  const { auth } = useStore();
  const storeDispatch = useStoreDispatch();

  const {
    contextHolder: modalPasswordChangeContextHolder,
    modalOpen: openChangePasswordModal,
  } = useModalPasswordChange();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [ openChangePasswordModal ]);

  const handleSave = useCallback(async (newValue: string) => {
    try {
      const user = await patchUser({
        userId: auth.user?.id || 0,
        userParams: { displayName: newValue },
      });
      storeDispatch({ type: StoreAction.SetCurrentUser, value: user });
      message.success(API_SUCCESS_MESSAGE);
    } catch (e) {
      message.error(API_ERROR_MESSAGE);
      handleError(e, { silent: true, type: ErrorType.Input });
    }
  }, [ auth.user, storeDispatch ]);

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip size={Size.ExtraLarge} userId={auth.user?.id} />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Username</label>
        <div className={css.info}>{auth.user?.username}</div>
      </div>
      <Divider />
      <div className={css.row}>
        <label>Display Name</label>
        <InlineEditor
          maxLength={CharLength.Limit32}
          pattern={new RegExp('^[a-z][a-z0-9\\s]*$', 'i')}
          placeholder="Add display name"
          value={auth.user?.displayName || ''}
          onSave={handleSave}
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
