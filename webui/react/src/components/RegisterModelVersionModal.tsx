import { Input, Modal, Select } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import handleError, { ErrorType } from 'ErrorHandler';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import { Metadata, ModelItem } from 'types';
import { isEqual } from 'utils/data';

import EditableMetadata from './Metadata/EditableMetadata';
import css from './RegisterModelVersionModal.module.scss';
import EditableTagList from './TagList';

const { Option } = Select;

interface Props {
  onClose?: () => void;
  visible?: boolean;
}

const RegisterModelVersionModal: React.FC<Props> = ({ visible = false, onClose }) => {
  const [ selectedModel, setSelectedModel ] = useState<string>();
  const [ models, setModels ] = useState<ModelItem[]>([]);
  const [ versionName, setVersionName ] = useState('');
  const [ modelDescription, setModelDescription ] = useState('');
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

  const updateModel = useCallback((value) => {
    setSelectedModel(value);
  }, []);

  const updateVersionName = useCallback((value) => {
    setVersionName(value);
  }, []);

  const updateModelDescription = useCallback((value) => {
    setModelDescription(value);
  }, []);

  const modelOptions = useMemo(() => {
    return models.map(model => model.name);
  }, [ models ]);

  const openDetails = useCallback(() => {
    setExpandDetails(true);
  }, []);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  useEffect(() => {
    fetchModels();
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
          <Select
            dropdownMatchSelectWidth={250}
            placeholder="Select a model..."
            onChange={updateModel}>
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
