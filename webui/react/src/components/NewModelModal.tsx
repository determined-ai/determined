import { Input, Modal } from 'antd';
import React, { useCallback, useState } from 'react';

import { useStore } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { postModel } from 'services/api';
import { Metadata } from 'types';

import EditableMetadata from './Metadata/EditableMetadata';
import css from './NewModelModal.module.scss';
import EditableTagList from './TagList';

interface Props {
  onClose?: () => void;
  visible?: boolean;
}

const NewModelModal: React.FC<Props> = ({ visible = false, onClose }: Props) => {
  const { auth: { user } } = useStore();
  const [ modelName, setModelName ] = useState('');
  const [ modelDescription, setModelDescription ] = useState('');
  const [ tags, setTags ] = useState<string[]>([]);
  const [ metadata, setMetadata ] = useState<Metadata>({});

  const createModel = useCallback(async () => {
    try {
      await postModel({
        description: modelDescription,
        labels: tags,
        metadata: metadata,
        name: modelName,
        username: user?.username,
      });
      onClose?.();
    } catch {
      handleError({ message: 'Unable to create model.', silent: true, type: ErrorType.Api });
    }
  }, [ metadata, tags, modelDescription, modelName, onClose, user?.username ]);

  const updateModelName = useCallback((value) => {
    setModelName(value);
  }, []);

  const updateModelDescription = useCallback((value) => {
    setModelDescription(value);
  }, []);

  return (
    <Modal
      okButtonProps={{ disabled: modelName === '' }}
      okText="Create Model"
      title="New Model"
      visible={visible}
      onCancel={onClose}
      onOk={createModel}>
      <div className={css.base}>
        <p className={css.directions}>
          Create a registered model to organize important checkpoints.
        </p>
        <div>
          <h2>Model name</h2>
          <Input value={modelName} onChange={updateModelName} />
        </div>
        <div>
          <h2>Description <span>(optional)</span></h2>
          <Input.TextArea value={modelDescription} onChange={updateModelDescription} />
        </div>
        <div>
          <h2>Metadata <span>(optional)</span></h2>
          <EditableMetadata editing={true} metadata={metadata} updateMetadata={setMetadata} />
        </div>
        <div>
          <h2>Tags <span>(optional)</span></h2>
          <EditableTagList tags={tags} onChange={setTags} />
        </div>
      </div>
    </Modal>
  );
};

export default NewModelModal;
