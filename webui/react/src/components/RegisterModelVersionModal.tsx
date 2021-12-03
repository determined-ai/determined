import { Input, Modal, notification, Select } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import handleError, { ErrorType } from 'ErrorHandler';
import { paths } from 'routes/utils';
import { getModels, postModelVersion } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { validateDetApiEnum } from 'services/utils';
import { Metadata, ModelItem } from 'types';
import { isEqual } from 'utils/data';

import Link from './Link';
import EditableMetadata from './Metadata/EditableMetadata';
import NewModelModal from './NewModelModal';
import css from './RegisterModelVersionModal.module.scss';
import EditableTagList from './TagList';
interface Props {
  checkpointUuid: string;
  onClose?: () => void;
  onCloseAll?: () => void;
  visible?: boolean;
}

const RegisterModelVersionModal: React.FC<Props> = (
  { checkpointUuid, visible = false, onClose, onCloseAll },
) => {
  const [ selectedModelId, setSelectedModelId ] = useState<number>();
  const [ models, setModels ] = useState<ModelItem[]>([]);
  const [ versionName, setVersionName ] = useState('');
  const [ versionDescription, setVersionDescription ] = useState('');
  const [ tags, setTags ] = useState<string[]>([]);
  const [ metadata, setMetadata ] = useState<Metadata>({});
  const [ expandDetails, setExpandDetails ] = useState(false);
  const [ canceler ] = useState(new AbortController());
  const [ showNewModelModal, setShowNewModelModal ] = useState(false);

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

  const selectedModelNumVersions = useMemo(() => {
    return models.find(model => model.id === selectedModelId)?.numVersions ?? 0;
  }, [ models, selectedModelId ]);

  const registerModelVersion = useCallback(async () => {
    if (!selectedModelId) return;
    try {
      const response = await postModelVersion({
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
      onCloseAll?.();
      if (!response) return;
      notification.open(
        {
          btn: null,
          description: (
            <div className={css.toast}>
              <p>{`"${versionName || `Version ${selectedModelNumVersions + 1}`}"`} registered</p>
              <Link path={paths.modelVersionDetails(selectedModelId, response.id)}>
              View Model Version
              </Link>
            </div>),
          message: '',
        },
      );
    } catch {
      handleError({
        message: 'Unable to register model version.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ checkpointUuid,
    metadata,
    tags,
    versionDescription,
    versionName,
    onCloseAll,
    selectedModelId,
    selectedModelNumVersions ]);

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

  const openDetails = useCallback(() => {
    setExpandDetails(true);
  }, []);

  const closeNewModelModal = useCallback(async (newModelId?: number) => {
    setShowNewModelModal(false);
    if (newModelId) {
      await fetchModels();
      setSelectedModelId(newModelId);
    }
  }, [ fetchModels ]);

  const launchNewModelModal = useCallback(() => {
    setShowNewModelModal(true);
  }, []);

  useEffect(() => {
    fetchModels();
  }, [ fetchModels ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <>
      <Modal
        okButtonProps={{ disabled: selectedModelId == null }}
        okText="Register Checkpoint"
        title="Register Checkpoint"
        visible={visible}
        onCancel={onClose}
        onOk={registerModelVersion}>
        <div className={css.base}>
          <p className={css.directions}>Save this checkpoint to the Model Registry</p>
          <div>
            <div className={css.selectModelRow}>
              <h2>Select Model</h2>
              <p onClick={launchNewModelModal}>New Model</p>
            </div>
            <Select
              dropdownMatchSelectWidth={250}
              optionFilterProp="label"
              options={modelOptions.map(option => (
                { label: option.name, value: option.id }))}
              placeholder="Select a model..."
              showSearch
              value={selectedModelId}
              onChange={updateModel}
            />
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
      <NewModelModal visible={showNewModelModal} onClose={closeNewModelModal} />
    </>
  );
};

export default RegisterModelVersionModal;
