import { Button, Divider } from 'antd';
import { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useCallback } from 'react';

import Avatar from 'components/Avatar';
import { useStore } from 'contexts/Store';
import useModalChangeName from 'hooks/useModal/UserSettings/useModalChangeName';
import useModalChangePassword from 'hooks/useModal/UserSettings/useModalChangePassword';

import useModal, { ModalHooks } from '../useModal';

import css from './useModalUserSettings.module.scss';

interface Props {
  modal: Omit<ModalStaticFunctions, 'warn'>
}

const UserSettings: React.FC<Props> = ({ modal }) => {
  const { auth } = useStore();

  const { modalOpen: openChangeDisplayNameModal } = useModalChangeName(modal);
  const { modalOpen: openChangePasswordModal } = useModalChangePassword(modal);

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
          <Avatar hideTooltip large userId={auth.user?.id} />
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

const useModalUserSettings = (modal: Omit<ModalStaticFunctions, 'warn'>): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ modal });

  const modalOpen = useCallback(() => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: <UserSettings modal={modal} />,
      icon: null,
      title: <h5>Account</h5>,
    });
  }, [ modal, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalUserSettings;
