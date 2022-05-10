import { Input, Modal } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { copyToClipboard } from 'shared/utils/dom';
import { ModelVersion } from 'types';

import CopyButton from './CopyButton';
import css from './DownloadModelModal.module.scss';

interface Props {
  modelVersion: ModelVersion;
  onClose: () => void;
  visible: boolean;
}

const DownloadModelPopover: React.FC<Props> = (
  { visible, modelVersion, onClose }: Props,
) => {

  const downloadCommand = useMemo(() => {
    return `det checkpoint download ${modelVersion.checkpoint.uuid}`;
  }, [ modelVersion.checkpoint.uuid ]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(downloadCommand);
  }, [ downloadCommand ]);

  return (
    <Modal
      footer={null}
      title="Download"
      visible={visible}
      onCancel={onClose}>
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
    </Modal>
  );
};

export default DownloadModelPopover;
