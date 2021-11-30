import { Input, Modal, Select } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import handleError, { ErrorType } from 'ErrorHandler';
import { getModels, postModelVersion } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import { Metadata, ModelItem } from 'types';
import { isEqual } from 'utils/data';

import EditableMetadata from './Metadata/EditableMetadata';
import css from './RegisterModelVersionModal.module.scss';
import EditableTagList from './TagList';

const { Option } = Select;

interface Props {
  checkpointUuid: string;
  onClose?: () => void;
  visible?: boolean;
}

const RegisterModelVersionModal: React.FC<Props> = (
  { checkpointUuid, visible = false, onClose },
) => {
  const [ selectedModelId, setSelectedModelId ] = useState<number>();
  const [ models, setModels ] = useState<ModelItem[]>([]);
  const [ versionName, setVersionName ] = useState('');
  const [ versionDescription, setVersionDescription ] = useState('');
  const [ tags, setTags ] = useState<string[]>([]);
  const [ metadata, setMetadata ] = useState<Metadata>({});
  const [ expandDetails, setExpandDetails ] = useState(false);
  const [ canceler ] = useState(new AbortController());

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels({
        orderBy: 'ORDER_BY_DESC',
        sortBy: validateDetApiEnum(
          V1GetModelsRequestSortBy,
          V1GetModelsRequestSortBy.LASTUPDATEDTIME,
        ),
      }, { signal: canceler.signal });
      setModels(prev => {
        if (isEqual(prev, response.models)) return prev;
        return response.models;
      });
    } catch(e) {
      handleError({ message: 'Unable to fetch models.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler.signal ]);

  const registerModelVersion = useCallback(async () => {
    if (!selectedModelId) return;
    try {
      await postModelVersion({
        body: {
          checkpointUuid,
          comment: versionDescription,
          labels: tags,
          metadata,
          modelId: selectedModelId,
          name: versionName,
        },
        modelId: selectedModelId,
      });
      onClose?.();
    } catch {
      handleError({ message: 'Unable to create model.', silent: true, type: ErrorType.Api });
    }
  }, [ checkpointUuid, metadata, tags, versionDescription, versionName, onClose, selectedModelId ]);

  const updateModel = useCallback((value) => {
    setSelectedModelId(value);
  }, []);

  const updateVersionName = useCallback((e) => {
    setVersionName(e.target.value);
  }, []);

  const updateVersionDescription = useCallback((e) => {
    setVersionDescription(e.target.value);
  }, []);

  const modelOptions = useMemo(() => {
    return models.map(model => ({ id: model.id, name: model.name }));
  }, [ models ]);

  const selectedModelNumVersions = useMemo(() => {
    return models.find(model => model.id === selectedModelId)?.numVersions ?? 0;
  }, [ models, selectedModelId ]);

  const openDetails = useCallback(() => {
    setExpandDetails(true);
  }, []);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  useEffect(() => {
    fetchModels();
  }, [ fetchModels ]);

  return (
    <Modal
      okButtonProps={{ disabled: selectedModelId == null }}
      okText="Add Model Version"
      title="Register Model"
      visible={visible}
      onCancel={onClose}
      onOk={registerModelVersion}>
      <div className={css.base}>
        <p className={css.directions}>Save this checkpoint to the Model Registry</p>
        <div>
          <h2>Select Model</h2>
          <Select
            dropdownMatchSelectWidth={250}
            placeholder="Select a model..."
            showSearch
            onChange={updateModel}>
            {modelOptions.map(option => (
              <Option key={option.id} value={option.id}>
                {option.name}
              </Option>))}
          </Select>
        </div>
        <div className={css.separator} />
        <div>
          <h2>Version Name</h2>
          <Input
            placeholder={`Version ${selectedModelNumVersions + 1}`}
            value={versionName}
            onChange={updateVersionName} />
        </div>
        <div>
          <h2>Description <span>(optional)</span></h2>
          <Input.TextArea value={versionDescription} onChange={updateVersionDescription} />
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

export default RegisterModelVersionModal;
