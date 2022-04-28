import { Button, Divider } from 'antd';
import { ModalStaticFunctions } from 'antd/es/modal/confirm';
import React, { useCallback, useState } from 'react';

import Avatar from 'components/Avatar';
import { useStore } from 'contexts/Store';
import useModalChangeName from './useModalChangeName';
import useModalChangePassword from './useModalChangePassword';
import { setUserImage } from 'services/api';

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
          setPreviewImage(miniCanvas.toDataURL('image/jpeg'));
          // modalClose();
        };
        img.src = String(reader.result);
      }
    };
    if (event && event.target && event.target.files) {
      reader.readAsDataURL(event.target.files[0]);
    }
  };

  const handleIconUploadClick = useCallback(async () => {
    if (!previewImage.length || !auth.user || !auth.user?.id) {
      return;
    }
    try {
      await setUserImage({
        image: previewImage.substring(previewImage.indexOf(",") + 1),
        userId: auth.user?.id,
      });
      setPreviewImage('');
    } catch (e) {

    }
  }, [ auth.user, previewImage ]);

  return (
    <div className={css.base}>
      <div className={css.field}>
        <span className={css.header}>Avatar</span>
        <span className={css.body}>
          {previewImage.length
            ? <img src={previewImage} height="64" width="64" style={{border: '1px solid #000'}}/>
            : <Avatar hideTooltip large userId={auth.user?.id} />
          }
          <input type="file" accept="image/png, image/jpeg" onChange={processProfilePic}/>
          <Button onClick={handleIconUploadClick}>
            Upload
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
