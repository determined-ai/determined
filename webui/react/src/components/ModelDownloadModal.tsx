import { Modal } from 'components/kit/Modal';
import CopyButton from 'shared/components/CopyButton';
import { copyToClipboard } from 'shared/utils/dom';
import { ModelVersion } from 'types';

import css from './ModelDownloadModal.module.scss';

interface Props {
  modelVersion: ModelVersion;
}

const ModelDownloadModal = ({ modelVersion }: Props): JSX.Element => {
  const downloadCommand = `det checkpoint download ${modelVersion?.checkpoint.uuid}`;
  const handleCopy = async () => await copyToClipboard(downloadCommand);

  return (
    <Modal size="medium" title="Download Model Command">
      <div className={css.base}>
        <div className={css.commandContainer}>
          <code className={css.codeSample}>{downloadCommand}</code>
          <div>
            <CopyButton onCopy={handleCopy} />
          </div>
        </div>
        <p className={css.bottomLine}>Copy/paste command into the Determined CLI</p>
      </div>
    </Modal>
  );
};

export default ModelDownloadModal;
