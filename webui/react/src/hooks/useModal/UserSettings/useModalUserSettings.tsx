import { Button, Divider } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Avatar from 'components/Avatar';
import { useStore } from 'contexts/Store';
import useModalChangeName from 'hooks/useModal/UserSettings/useModalChangeName';
import useModalChangePassword from 'hooks/useModal/UserSettings/useModalChangePassword';

import useModal, { ModalHooks } from '../useModal';

import css from './useModalUserSettings.module.scss';

interface UserValues {
  displayName: string;
  username: string;
}

const useModalUserSettings = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();
  const { auth } = useStore();

  const [ userValues, setUserValues ] = useState({
    displayName: '',
    username: '',
  });

  const [ modalProps, setModalProps ] = useState({
    className: css.noFooter,
    closable: true,
    icon: null,
    title: 'Account',
  });

  const { modalOpen: openChangeDisplayNameModal } = useModalChangeName();
  const { modalOpen: openChangePasswordModal } = useModalChangePassword();

  const handlePasswordClick = useCallback(() => {
    openChangePasswordModal();
  }, [ openChangePasswordModal ]);

  const handleDisplayNameClick = useCallback(() => {
    openChangeDisplayNameModal();
  }, [ openChangeDisplayNameModal ]);

  const getModalContent = useCallback((userValues: UserValues): React.ReactNode => {
    return (
      <div className={css.base}>
        <div className={css.field}>
          <span className={css.header}>Avatar</span>
          <span className={css.body}>
            <Avatar hideTooltip large name={userValues.displayName || userValues.username} />
          </span>
          <Divider />
        </div>
        <div className={css.field}>
          <span className={css.header}>Display name</span>
          <span className={css.body}>
            <span>{userValues.displayName}</span>
            <Button
              onClick={handleDisplayNameClick}>
              Change name
            </Button>
          </span>
          <Divider />
        </div>
        <div className={css.field}>
          <span className={css.header}>Username</span>
          <span className={css.body}>{userValues.username}</span>
          <Divider />
        </div>
        <div className={css.field}>
          <span className={css.header}>Password</span>
          <span className={css.body}>
            <Button
              onClick={handlePasswordClick}>
              Change password
            </Button>
          </span>
        </div>
      </div>
    );
  }, [ handleDisplayNameClick, handlePasswordClick ]);

  useEffect(() => {
    setUserValues({
      displayName: auth.user?.displayName || '',
      username: auth.user?.username || 'Anonymous',
    });
  }, [ auth ]);

  useEffect(() => {
    setModalProps(modalProps => {
      return { ...modalProps, content: getModalContent(userValues) };
    });
  }, [ userValues, getModalContent ]);

  useEffect(() => {
    // update modal
    if (modalRef.current) openOrUpdate(modalProps);
  }, [ modalProps, openOrUpdate, modalRef ]);

  const modalOpen = useCallback(() => {
    // open modal
    openOrUpdate({ ...modalProps });
  }, [ modalProps, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalUserSettings;
