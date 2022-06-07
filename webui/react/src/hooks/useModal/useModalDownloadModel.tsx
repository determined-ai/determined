import { Input, ModalFuncProps } from 'antd';
import React, { useCallback, useMemo } from 'react';

import CopyButton from 'components/CopyButton';
import { copyToClipboard } from 'shared/utils/dom';
import { ModelVersion } from 'types';

import useModal, { ModalHooks as Hooks } from './useModal';
import css from './useModalDownloadModel.module.scss';

interface Props {
  modelVersion: ModelVersion;
  onClose?: () => void;
}

export interface ShowModalProps {
  initialModalProps?: ModalFuncProps;
}

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (props: ShowModalProps) => void;
}

const useModalDownloadModel = ({ modelVersion, onClose }: Props): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose });

  const downloadCommand = useMemo(() => {
    return `det checkpoint download ${modelVersion.checkpoint.uuid}`;
  }, [ modelVersion.checkpoint.uuid ]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(downloadCommand);
  }, [ downloadCommand ]);

  const modalContent = useMemo(() => {
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
  }, [ downloadCommand, handleCopy ]);

  const modalProps: ModalFuncProps = useMemo(() => {
    return {
      bodyStyle: { padding: 0 },
      closable: true,
      content: modalContent,
      footer: null,
      icon: null,
      maskClosable: true,
      title: 'Download',
    };
  }, [ modalContent ]);

  const modalOpen = useCallback(({ initialModalProps }: ShowModalProps) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalDownloadModel;
