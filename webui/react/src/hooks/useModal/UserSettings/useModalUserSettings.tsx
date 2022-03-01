import { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useCallback } from 'react';

import UserSettings from 'components/UserSettings';

import useModal, { ModalHooks } from '../useModal';

import css from './useModalUserSettings.module.scss';

const useModalUserSettings = (modal: Omit<ModalStaticFunctions, 'warn'>): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ modal });

  const modalOpen = useCallback(() => {
    openOrUpdate({
      className: css.noFooter,
      closable: true,
      content: <UserSettings />,
      icon: null,
      title: 'Account',
    });
  }, [ openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalUserSettings;
