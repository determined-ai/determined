import { Input, Modal } from 'antd';
import React, { useCallback, useState } from 'react';
import { debounce } from 'throttle-debounce';

import { useStore } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { getModels, postModel } from 'services/api';
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
  const [ isNameUnique, setIsNameUnique ] = useState(true);
  const [ modelDescription, setModelDescription ] = useState('');
  const [ tags, setTags ] = useState<string[]>([]);
  const [ metadata, setMetadata ] = useState<Metadata>({});
  const [ expandDetails, setExpandDetails ] = useState(false);

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

  const findIsNameUnique = useCallback(async (name) => {
    const modelsList = await getModels({ name: name });
    setIsNameUnique(modelsList.pagination.total === 0 || name === '');
  }, []);

  const updateModelName = useCallback((e) => {
    setModelName(e.target.value);
    debounce(250, () => findIsNameUnique(e.target.value))();
  }, [ findIsNameUnique ]);

  const updateModelDescription = useCallback((e) => {
    setModelDescription(e.target.value);
  }, []);

  const openDetails = useCallback(() => {
    setExpandDetails(true);
  }, []);

  return (
    <Modal
      okButtonProps={{ disabled: modelName === '' || !isNameUnique }}
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
          {!isNameUnique &&
          <p className={css.uniqueWarning}>A model with this name already exists</p>}
        </div>
        <div>
          <h2>Description <span>(optional)</span></h2>
          <Input.TextArea value={modelDescription} onChange={updateModelDescription} />
        </div>
        {expandDetails ?
          <>
            <div>
              <h2>Metadata <span>(optional)</span></h2>
              <EditableMetadata editing={true} metadata={metadata} updateMetadata={setMetadata} />
            </div>
            <div>
              <h2>Tags <span>(optional)</span></h2>
              <EditableTagList tags={tags} onChange={setTags} />
            </div>
          </> :
          <p className={css.expandDetails} onClick={openDetails}>Add More Details...</p>}
      </div>
    </Modal>
  );
};

export default NewModelModal;
