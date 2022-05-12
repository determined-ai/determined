import { Input, Modal, ModalFuncProps, notification, Select } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import EditableTagList from 'components/TagList';
import { paths } from 'routes/utils';
import { getModels, postModelVersion } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { isEqual } from 'shared/utils/data';
import { Metadata, ModelItem } from 'types';
import handleError from 'utils/error';

import { ErrorType } from '../shared/utils/error';
import { validateDetApiEnum } from '../shared/utils/service';

import css from './useRegisterCheckpointModal.module.scss';

export interface ShowRegisterCheckpointProps {
  checkpointUuid: string;
  selectedModelName?: string;
}

interface ModalState {
  checkpointUuid?: string;
  expandDetails: boolean;
  metadata: Metadata;
  selectedModelName?: string;
  tags: string[];
  versionDescription: string;
  versionName: string;
  visible: boolean;

}

interface ModalHooks {
  showModal: (props: ShowRegisterCheckpointProps) => void;
}

const useRegisterCheckpointModal = (onClose?: (checkpointUuid?: string) => void): ModalHooks => {
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const [ models, setModels ] = useState<ModelItem[]>([]);
  const [ canceler ] = useState(new AbortController());
  const [ modalState, setModalState ] = useState<ModalState>({
    expandDetails: false,
    metadata: {},
    tags: [],
    versionDescription: '',
    versionName: '',
    visible: false,
  });

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels({
        archived: false,
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
      handleError(e, {
        publicSubject: 'Unable to fetch models.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ canceler.signal ]);

  useEffect(() => {
    fetchModels();
  }, [ fetchModels ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const showModal = useCallback((
    { checkpointUuid, selectedModelName }: ShowRegisterCheckpointProps,
  ) => {
    fetchModels();
    setModalState({
      checkpointUuid,
      expandDetails: false,
      metadata: {},
      selectedModelName,
      tags: [],
      versionDescription: '',
      versionName: '',
      visible: true,
    });
  }, [ fetchModels ]);

  const closeModal = useCallback(() => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
  }, []);

  const handleCancel = useCallback(() => {
    closeModal();
  }, [ closeModal ]);

  const selectedModelNumVersions = useMemo(() => {
    return models.find(model => model.name === modalState.selectedModelName)?.numVersions ?? 0;
  }, [ models, modalState.selectedModelName ]);

  const registerModelVersion = useCallback(async (state: ModalState) => {
    const {
      selectedModelName, versionDescription, tags,
      metadata, versionName, checkpointUuid,
    } = state;
    if (!selectedModelName || !checkpointUuid) return;
    try {
      const response = await postModelVersion({
        body: {
          checkpointUuid,
          comment: versionDescription,
          labels: tags,
          metadata,
          modelName: selectedModelName,
          name: versionName,
        },
        modelName: selectedModelName,
      });
      if (!response) return;
      closeModal();
      notification.open(
        {
          btn: null,
          description: (
            <div className={css.toast}>
              <p>{`"${versionName || `Version ${selectedModelNumVersions + 1}`}"`} registered</p>
              <Link path={paths.modelVersionDetails(selectedModelName, response.id)}>
                View Model Version
              </Link>
            </div>),
          message: '',
        },
      );
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to register checkpoint.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ closeModal, selectedModelNumVersions ]);

  const handleOk = useCallback(async (state: ModalState) => {
    if (!modalRef.current) return Promise.reject();
    await registerModelVersion(state);
  }, [ registerModelVersion ]);

  const updateModel = useCallback((value) => {
    setModalState(prev => ({ ...prev, selectedModelName: value }));
  }, []);

  const updateVersionName = useCallback((e) => {
    setModalState(prev => ({ ...prev, versionName: e.target.value }));
  }, []);

  const updateVersionDescription = useCallback((e) => {
    setModalState(prev => ({ ...prev, versionDescription: e.target.value }));
  }, []);

  const modelOptions = useMemo(() => {
    return models.map(model => ({ id: model.id, name: model.name }));
  }, [ models ]);

  const openDetails = useCallback(() => {
    setModalState(prev => ({ ...prev, expandDetails: true }));
  }, []);

  const updateMetadata = useCallback((value) => {
    setModalState(prev => ({ ...prev, metadata: value }));
  }, []);

  const updateTags = useCallback((value) => {
    setModalState(prev => ({ ...prev, tags: value }));
  }, []);

  const launchNewModelModal = useCallback((state: ModalState) => {
    const { checkpointUuid } = state;
    closeModal();
    onClose?.(checkpointUuid);
  }, [ closeModal, onClose ]);

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
    const {
      selectedModelName, versionDescription,
      tags, metadata, versionName, expandDetails,
    } = state;

    // We always render the form regardless of mode to provide a reference to it.
    return (
      <div className={css.base}>
        <p className={css.directions}>Save this checkpoint to the Model Registry</p>
        <div>
          <div className={css.selectModelRow}>
            <h2>Select Model</h2>
            <p onClick={() => launchNewModelModal(state)}>New Model</p>
          </div>
          <Select
            optionFilterProp="label"
            options={modelOptions.map(option => (
              { label: option.name, value: option.name }))}
            placeholder="Select a model..."
            showSearch
            style={{ width: '100%' }}
            value={selectedModelName}
            onChange={updateModel}
          />
        </div>
        {selectedModelName && (
          <>
            <div className={css.separator} />
            <div>
              <h2>Version Name</h2>
              <Input
                placeholder={`Version ${selectedModelNumVersions + 1}`}
                value={versionName}
                onChange={updateVersionName}
              />
            </div>
            <div>
              <h2>Description <span>(optional)</span></h2>
              <Input.TextArea value={versionDescription} onChange={updateVersionDescription} />
            </div>
            {expandDetails ? (
              <>
                <div>
                  <h2>Metadata <span>(optional)</span></h2>
                  <EditableMetadata
                    editing={true}
                    metadata={metadata}
                    updateMetadata={updateMetadata}
                  />
                </div>
                <div>
                  <h2>Tags <span>(optional)</span></h2>
                  <EditableTagList tags={tags} onChange={updateTags} />
                </div>
              </>
            ) :
              <p className={css.expandDetails} onClick={openDetails}>Add More Details...</p>}
          </>
        )}
      </div>
    );
  }, [ launchNewModelModal,
    modelOptions,
    openDetails,
    selectedModelNumVersions,
    updateMetadata,
    updateModel,
    updateTags,
    updateVersionDescription,
    updateVersionName ]);

  const generateModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    const { selectedModelName } = state;

    const modalProps = {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: generateModalContent(state),
      icon: null,
      maskClosable: true,
      okButtonProps: { disabled: selectedModelName == null },
      okText: 'Register Checkpoint',
      onCancel: handleCancel,
      onOk: () => handleOk(state),
      title: 'Register Checkpoint',
    };

    return modalProps;
  }, [ generateModalContent, handleCancel, handleOk ]);

  // Detect modal state change and update.
  useEffect(() => {
    if (!modalState.visible) return;

    const modalProps = generateModalProps(modalState);
    if (modalRef.current) {
      modalRef.current.update(prev => ({ ...prev, ...modalProps }));
    } else {
      modalRef.current = Modal.confirm(modalProps);
    }
  }, [ generateModalProps, modalState ]);

  // When the component using the hook unmounts, remove the modal automatically.
  useEffect(() => {
    return () => {
      if (!modalRef.current) return;
      modalRef.current.destroy();
      modalRef.current = undefined;
    };
  }, []);

  return { showModal };
};

export default useRegisterCheckpointModal;
