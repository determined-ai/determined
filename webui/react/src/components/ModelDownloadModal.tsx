import ClipboardButton from 'components/kit/ClipboardButton';
import { Modal } from 'components/kit/Modal';
import css from 'components/ModelDownloadModal.module.scss';
import { ModelVersion } from 'types';

interface Props {
  modelVersion: ModelVersion;
}

const ModelDownloadModal = ({ modelVersion }: Props): JSX.Element => {
  const downloadCommand = `det checkpoint download ${modelVersion?.checkpoint.uuid}`;

  return (
    <Modal size="medium" title="Download Model Command">
      <div className={css.base}>
        <div className={css.commandContainer}>
          <code className={css.codeSample}>{downloadCommand}</code>
          <div>
            <ClipboardButton getContent={() => downloadCommand} />
          </div>
        </div>
        <p className={css.bottomLine}>Copy/paste command into the Determined CLI</p>
      </div>
    </Modal>
  );
};

export default ModelDownloadModal;
