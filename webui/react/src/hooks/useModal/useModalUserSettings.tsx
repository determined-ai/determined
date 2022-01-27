import React from 'react';

import UserSettings from 'components/UserSettings';
import { useStore } from 'contexts/Store';

import useModal, { ModalHooks } from './useModal';
import css from './useModalUserSettings.module.scss';

const useModalUserSettings = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const { auth } = useStore();
  const username = auth.user?.username || 'Anonymous';

  const getModalContent = () => {
    return <UserSettings username={username} />;
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
