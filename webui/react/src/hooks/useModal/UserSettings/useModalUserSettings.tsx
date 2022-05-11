import { Button, Divider, Upload } from 'antd';
import { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useCallback, useState } from 'react';

import Avatar from 'components/Avatar';
import { useStore } from 'contexts/Store';
import { setUserImage } from 'services/api';
import handleError from 'utils/error';

import useModal, { ModalHooks } from '../useModal';

import useModalChangeName from './useModalChangeName';
import useModalChangePassword from './useModalChangePassword';
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

  const [ previewImage, setPreviewImage ] = useState(false);
  const handleIconUpload = useCallback((file) => {
    const reader = new FileReader();
    reader.onloadend = async () => {
      try {
        const readerResult = String(reader.result);
        await setUserImage({
          image: readerResult.substring(readerResult.indexOf(',') + 1),
          userId: auth.user?.id || 0,
        });
      } catch (e) {
        handleError(e);
      }
    };
    reader.readAsDataURL(file);
    setPreviewImage(true);
    return false;
  }, [ auth.user ]);

  const handleRemoveIcon = useCallback(() => {
    setPreviewImage(false);
    setUserImage({
      // need to send SOME data for Proto to accept content as bytes type
      image: '00000000',
      userId: auth.user?.id || 0,
    });
  }, [ auth.user ]);

  return (
    <div className={css.base}>
      <div className={css.field}>
        <span className={css.header}>Avatar</span>
        <span className={css.body}>
          {previewImage ? '' : <Avatar hideTooltip large userId={auth.user?.id} />}
          <Upload
            accept="image/png, image/jpeg"
            beforeUpload={handleIconUpload}
            listType="picture-card"
            maxCount={1}
            onRemove={handleRemoveIcon}>
            <Button>
              Upload
            </Button>
          </Upload>
          <Button onClick={handleRemoveIcon}>
            Clear
          </Button>
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
