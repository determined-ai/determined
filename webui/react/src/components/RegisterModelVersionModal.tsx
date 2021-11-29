import { Input, Modal, Select } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import css from './RegisterModelVersionModal.module.scss';
import EditableTagList from './TagList';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  visible?: boolean;
}

const RegisterModelVersionModal: React.FC<Props> = ({ visible = false, onClose }) => {
  const [ model, setModel ] = useState<string>();
  const [ versionName, setVersionName ] = useState('');
  const [ modelDescription, setModelDescription ] = useState('');
  const [ tags, setTags ] = useState<string[]>([]);

  const updateModel = useCallback((value) => {
    setModel(value);
  }, []);

  const updateVersionName = useCallback((value) => {
    setVersionName(value);
  }, []);

  const updateModelDescription = useCallback((value) => {
    setModelDescription(value);
  }, []);

  const modelOptions = useMemo(() => {
    return [ '' ];
  }, []);

  return (
    <Modal
      okButtonProps={{ disabled: versionName === '' }}
      okText="Add Model Version"
      title="Register Model"
      visible={visible}
      onCancel={onClose}>
      <div className={css.base}>
        <p className={css.directions}>Save this checkpoint to the Model Registry</p>
        <div>
          <h2>Select Model</h2>
          <Select placeholder="Select a model..." onChange={updateModel}>
            {modelOptions.map(option => (
              <Option key={option} value={option}>
                {option === '' ?
                  'New Model' :
                  option}
              </Option>))}
          </Select>
        </div>
        <div>
          <h2>Version Name</h2>
          <Input value={versionName} onChange={updateVersionName} />
        </div>
        <div>
          <h2>Description <span>(optional)</span></h2>
          <Input.TextArea value={modelDescription} onChange={updateModelDescription} />
        </div>
        <div>
          <h2>Metadata <span>(optional)</span></h2>
          <p>Placeholder</p>
        </div>
        <div>
          <h2>Tags <span>(optional)</span></h2>
          <EditableTagList tags={tags} onChange={setTags} />
        </div>
      </div>
    </Modal>
  );
};

export default RegisterModelVersionModal;
