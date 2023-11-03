import CodeSample from 'determined-ui/CodeSample';
import { Modal } from 'determined-ui/Modal';

import { ModelVersion } from 'types';

interface Props {
  modelVersion: ModelVersion;
}

const ModelDownloadModal = ({ modelVersion }: Props): JSX.Element => {
  const downloadCommand = `det checkpoint download ${modelVersion?.checkpoint.uuid}`;

  return (
    <Modal size="medium" title="Download Model Command">
      <div>
        <CodeSample text={downloadCommand} />
        <p>Copy/paste command into the Determined CLI</p>
      </div>
    </Modal>
  );
};

export default ModelDownloadModal;
