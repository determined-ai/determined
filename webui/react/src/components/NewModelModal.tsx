import { Input, Modal } from 'antd';
import React, { useCallback, useState } from 'react';

import EditableTagList from './TagList';

interface Props {
  onClose?: () => void;
  visible?: boolean;
}

const NewModelModal: React.FC<Props> = ({ visible = false, onClose }: Props) => {
  const [ modelName, setModelName ] = useState('');
  const [ modelDescription, setModelDescription ] = useState('');
  const [ tags, setTags ] = useState<string[]>([]);

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
      onCancel={onClose}>
      <p>Create a registered model to organize important checkpoints.</p>
      <h2>Model name</h2>
      <Input value={modelName} onChange={updateModelName} />
      <h2>Description <span>(optional)</span></h2>
      <Input.TextArea value={modelDescription} onChange={updateModelDescription} />
      <h2>Metadata <span>(optional)</span></h2>
      <p>Placeholder</p>
      <h2>Tags <span>(optional)</span></h2>
      <EditableTagList tags={tags} onChange={setTags} />
    </Modal>
  );
};

export default NewModelModal;
