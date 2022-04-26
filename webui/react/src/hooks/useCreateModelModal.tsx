import { Input, Modal, ModalFuncProps, notification } from 'antd';
import { ModalFunc } from 'antd/es/modal/confirm';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { debounce } from 'throttle-debounce';

import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import EditableTagList from 'components/TagList';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import { getModels, postModel } from 'services/api';
import { Metadata } from 'types';
import handleError, { ErrorType } from 'utils/error';

import css from './useCreateModelModal.module.scss';
import useRegisterCheckpointModal from './useRegisterCheckpointModal';

export interface ShowCreateModelProps {
  checkpointUuid?: string;
}

interface ModalState {
  checkpointUuid?: string;
  expandDetails: boolean;
  isNameUnique?: boolean;
  metadata: Metadata;
  modelDescription: string;
  modelName: string;
  tags: string[];
  visible: boolean;
}

interface ModalHooks {
  showModal: (props: ShowCreateModelProps) => void;
}

const useCreateModelModal = (): ModalHooks => {
  const { showModal: showRegisterCheckpointModal } = useRegisterCheckpointModal();
  const modalRef = useRef<ReturnType<ModalFunc>>();
  const { auth: { user } } = useStore();
  const [ modalState, setModalState ] = useState<ModalState>({
    expandDetails: false,
    isNameUnique: true,
    metadata: {},
    modelDescription: '',
    modelName: '',
    tags: [],
    visible: false,
  });

  const showModal = useCallback(({ checkpointUuid }: ShowCreateModelProps) => {
    setModalState({
      checkpointUuid,
      expandDetails: false,
      isNameUnique: true,
      metadata: {},
      modelDescription: '',
      modelName: '',
      tags: [],
      visible: true,
    });
  }, []);

  const closeModal = useCallback(() => {
    if (!modalRef.current) return;
    modalRef.current.destroy();
    modalRef.current = undefined;
  }, []);

  const handleCancel = useCallback(() => {
    if (!modalRef.current) return;
    closeModal();
  }, [ closeModal ]);

  const createModel = useCallback(async (state: ModalState) => {
    const { checkpointUuid, modelDescription, tags, metadata, modelName } = state;
    try {
      const response = await postModel({
        description: modelDescription,
        labels: tags,
        metadata: metadata,
        name: modelName,
      });
      if (!response?.id) return;
      closeModal();
      if (checkpointUuid){
        showRegisterCheckpointModal({ checkpointUuid, selectedModelName: response.name });
      }
      notification.open({
        btn: null,
        description: (
          <div className={css.toast}>
            <p>{`"${modelName}"`} created</p>
            <Link path={paths.modelDetails(response.name)}>
              View Model
            </Link>
          </div>),
        message: '',
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to create model.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ closeModal, showRegisterCheckpointModal, user?.id ]);

  const handleOk = useCallback(async (state: ModalState) => {
    if (!modalRef.current) return Promise.reject();
    await createModel(state);
  }, [ createModel ]);

  const findIsNameUnique = useCallback(async (name) => {
    const modelsList = await getModels({ name: name });
    setModalState(prev => ({
      ...prev,
      isNameUnique: modelsList.pagination.total === 0 || name === '',
    }));
  }, []);

  const updateModelName = useCallback((e) => {
    setModalState(prev => ({ ...prev, modelName: e.target.value }));
    debounce(250, () => findIsNameUnique(e.target.value))();
  }, [ findIsNameUnique ]);

  const updateModelDescription = useCallback((e) => {
    setModalState(prev => ({ ...prev, modelDescription: e.target.value }));
  }, []);

  const openDetails = useCallback(() => {
    setModalState(prev => ({ ...prev, expandDetails: true }));
  }, []);

  const updateMetadata = useCallback((value) => {
    setModalState(prev => ({ ...prev, metadata: value }));
  }, []);

  const updateTags = useCallback((value) => {
    setModalState(prev => ({ ...prev, tags: value }));
  }, []);

  const generateModalContent = useCallback((state: ModalState): React.ReactNode => {
    const {
      modelDescription, isNameUnique,
      tags, metadata, modelName, expandDetails,
    } = state;

    // We always render the form regardless of mode to provide a reference to it.
    return (
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
      </div>
    );
  }, [ openDetails,
    updateMetadata,
    updateTags,
    updateModelDescription,
    updateModelName ]);

  const generateModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    const { modelName, isNameUnique } = state;

    const modalProps = {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: generateModalContent(state),
      icon: null,
      maskClosable: true,
      okButtonProps: { disabled: modelName === '' || !isNameUnique },
      okText: 'Create Model',
      onCancel: handleCancel,
      onOk: () => handleOk(state),
      title: 'Create Model',
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

export default useCreateModelModal;
