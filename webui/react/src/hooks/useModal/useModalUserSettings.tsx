import { Button } from 'antd';
import React, { useCallback } from 'react';

import Avatar from 'components/Avatar';
import { useStore } from 'contexts/Store';
import useModalChangePassword from 'hooks/useModal/useModalChangePassword';

import useModal, { ModalHooks } from './useModal';
import css from './useModalUserSettings.module.scss';

const useModalUserSettings = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const { auth } = useStore();
  const username = auth.user?.username || 'Anonymous';

  const { modalOpen: openChangePasswordModal } = useModalChangePassword();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [ openChangePasswordModal ]);

  const getModalContent = () => {
    return (
      <div className={css.base}>
        <div className={css.field}>
          <span className={css.label}>Avatar</span>
          <span className={css.value}>
            <Avatar hideTooltip large name={username} />
          </span>
        </div>
        <div className={css.field}>
          <span className={css.label}>Username</span>
          <span className={css.value}>{username}</span>
        </div>
        <div className={css.field}>
          <span className={css.label}>Password</span>
          <span className={css.value}>
            <Button onClick={handlePasswordClick}>Change password</Button>
          </span>
        </div>
      </div>
    );
  };

  const modalOpen = () => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: getModalContent(),
      icon: null,
      title: 'Account',
    });
  };

  return { modalClose, modalOpen, modalRef };
};

export default useModalUserSettings;
