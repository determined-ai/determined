import { Button, Divider } from 'antd';
import React, { useCallback } from 'react';

import Avatar from 'components/Avatar';
import { useStore } from 'contexts/Store';
import useModalChangeName from 'hooks/useModal/UserSettings/useModalChangeName';
import useModalChangePassword from 'hooks/useModal/UserSettings/useModalChangePassword';

import css from './UserSettings.module.scss';

const UserSettings: React.FC = () => {
  const { auth } = useStore();

  const { modalOpen: openChangeDisplayNameModal } = useModalChangeName();
  const { modalOpen: openChangePasswordModal } = useModalChangePassword();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [ openChangePasswordModal ]);

  const handleDisplayNameClick = useCallback(() => {
    openChangeDisplayNameModal();
  }, [ openChangeDisplayNameModal ]);

  return (
    <div className={css.base}>
      <div className={css.field}>
        <span className={css.header}>Avatar</span>
        <span className={css.body}>
          <Avatar hideTooltip large username={auth.user?.username} />
        </span>
        <Divider />
      </div>
      <div className={css.field}>
        <span className={css.header}>Display name</span>
        <span className={css.body}>
          <span>{auth.user?.displayName}</span>
          <Button onClick={handleDisplayNameClick}>
            Change name
          </Button>
        </span>
        <Divider />
      </div>
      <div className={css.field}>
        <span className={css.header}>Username</span>
        <span className={css.body}>{auth.user?.username}</span>
        <Divider />
      </div>
      <div className={css.field}>
        <span className={css.header}>Password</span>
        <span className={css.body}>
          <Button onClick={handlePasswordClick}>
            Change password
          </Button>
        </span>
      </div>
    </div>
  );
};

export default UserSettings;
