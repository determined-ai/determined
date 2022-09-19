import { Input } from 'antd';
import React, { useCallback } from 'react';

import CopyButton from 'shared/components/CopyButton';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { copyToClipboard } from 'shared/utils/dom';
import { ModelVersion } from 'types';

import css from './useModalModelDownload.module.scss';

interface Props {
  onClose?: (reason?: ModalCloseReason) => void;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (version: ModelVersion) => void;
}

const useModalModelDownload = ({ onClose }: Props = {}): ModalHooks => {
  const { modalOpen: openOrUpdate, ...modalHook } = useModal({ onClose });

  const getModalContent = useCallback((version: ModelVersion) => {
    const downloadCommand = `det checkpoint download ${version?.checkpoint.uuid}`;
    const handleCopy = async () => await copyToClipboard(downloadCommand);
    return (
      <div className={css.base}>
        <div className={css.topLine}>
          <p>Download Model Command</p>
          <CopyButton onCopy={handleCopy} />
        </div>
        <Input
          className={css.codeSample}
          value={downloadCommand}
        />
        <p className={css.bottomLine}>
          Copy/paste command into the Determined CLI
        </p>
      </div>
    );
  }, []);

  const getModalProps = useCallback((version: ModelVersion) => {
    return {
      cancelText: 'Okay',
      closable: true,
      content: getModalContent(version),
      footer: null,
      icon: null,
      okButtonProps: { style: { display: 'none' } },
      title: 'Download',
    };
  }, [ getModalContent ]);

  const modalOpen = useCallback((version: ModelVersion) => {
    openOrUpdate(getModalProps(version));
  }, [ getModalProps, openOrUpdate ]);

  return { modalOpen, ...modalHook };
};

export default useModalModelDownload;
