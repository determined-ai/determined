import { Button, Divider } from 'antd';
import React, { useCallback } from 'react';

import Avatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import useModalNameChange from 'hooks/useModal/UserSettings/useModalNameChange';
import useModalPasswordChange from 'hooks/useModal/UserSettings/useModalPasswordChange';

import css from './SettingsAccount.module.scss';

const SettingsAccount: React.FC = () => {
  const { auth } = useStore();
  const {
    contextHolder: modalNameChangeContextHolder,
    modalOpen: openChangeDisplayNameModal,
  } = useModalNameChange();
  const {
    contextHolder: modalPasswordChangeContextHolder,
    modalOpen: openChangePasswordModal,
  } = useModalPasswordChange();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [ openChangePasswordModal ]);

  const handleDisplayNameClick = useCallback(() => {
    openChangeDisplayNameModal();
  }, [ openChangeDisplayNameModal ]);

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip large userId={auth.user?.id} />
      </div>
      <Divider />
      <div className={css.row}>
        <div className={css.info}>
          <label>Display Name</label>
          <span>{auth.user?.displayName}</span>
        </div>
        <Button onClick={handleDisplayNameClick}>
          Change name
        </Button>
      </div>
      <Divider />
      <div className={css.row}>
        <div className={css.info}>
          <label>Username</label>
          <span>{auth.user?.username}</span>
        </div>
      </div>
      <Divider />
      <div className={css.row}>
        <label>Password</label>
        <Button onClick={handlePasswordClick}>
          Change password
        </Button>
      </div>
      {modalNameChangeContextHolder}
      {modalPasswordChangeContextHolder}
    </div>
  );
};

export default SettingsAccount;
