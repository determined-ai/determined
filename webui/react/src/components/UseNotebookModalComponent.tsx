import CodeSample from 'hew/CodeSample';
import { Modal } from 'hew/Modal';
import React, { useMemo } from 'react';

import { ModelVersion } from 'types';

import css from './UseNotebookModalComponent.module.scss';

interface Props {
  modelVersion: ModelVersion;
  onClose: () => void;
}

const UseNotebookModalComponent: React.FC<Props> = ({ modelVersion, onClose }: Props) => {
  const referenceText = useMemo(() => {
    const escapedModelName = modelVersion.model.name.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
    return `import determined as det
from determined.experimental import client

model_entry = client.get_model("${escapedModelName}")
version = model_entry.get_version(${modelVersion.version})
ckpt = version.checkpoint
path = ckpt.download()

# Load a PyTorchTrial from a checkpoint:
from determined import pytorch
my_trial = \\
    pytorch.load_trial_from_checkpoint_path(path)

# Load a Keras model from TFKerasTrial checkpoint:
from determined import keras
model = keras.load_model_from_checkpoint_path(path)

# Import your checkpointed code:
with det.import_from_path(path + "/code"):
    import my_model_def as ckpt_model_def
`;
  }, [modelVersion]);

  return (
    <Modal
      submit={{
        handleError: () => {},
        handler: onClose,
        text: 'Close',
      }}
      title="Use in Notebook"
      onClose={onClose}>
      <div className={css.topLine}>
        <p>Reference this model in a notebook</p>
      </div>
      <CodeSample text={referenceText} />
      <p>Copy/paste code into a notebook cell</p>
    </Modal>
  );
};

export default UseNotebookModalComponent;
