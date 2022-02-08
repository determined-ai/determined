import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

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

  const [previewImage, setPreviewImage] = useState("");

  const processProfilePic = (event: React.ChangeEvent<HTMLInputElement>) => {
    const reader = new FileReader();
    reader.onloadend = () => {
      const miniCanvas = document.createElement("canvas");
      const squareSize = 128; // 64px with support for retina+ screens
      miniCanvas.width = squareSize;
      miniCanvas.height = squareSize;
      const ctx = miniCanvas.getContext("2d");
      if (ctx && reader.result) {
        const img = new Image();
        img.onload = () => {
          let offsetX = 0, offsetY = 0, width = squareSize, height = squareSize;
          let scale = squareSize / Math.max(img.naturalWidth, img.naturalHeight);
          if (img.naturalWidth > img.naturalHeight) {
            height = Math.round(scale * img.naturalHeight);
            offsetY = (squareSize - height) / 2;
          } else if (img.naturalHeight > img.naturalWidth) {
            width = Math.round(scale * img.naturalWidth);
            offsetX = (squareSize - width) / 2;
          }
          ctx.drawImage(img, offsetX, offsetY, width, height);
          setPreviewImage(miniCanvas.toDataURL());
          modalClose();
        };
        img.src = String(reader.result);
      }
    };
    if (event && event.target && event.target.files) {
      reader.readAsDataURL(event.target.files[0]);
    }
  };

  const getModalContent = () => {
    return (
      <div className={css.base}>
        <div className={css.field}>
          <span className={css.label}>Avatar</span>
          <span className={css.value}>
            <Avatar hideTooltip large name={username} />
            <img src={previewImage} height="64" width="64" style={{border: '1px solid #000'}}/>
            <input type="file" accept="image/png, image/jpeg" onChange={processProfilePic}/>
            <button>Save</button>
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
