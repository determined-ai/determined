import { Input, ModalFuncProps, notification } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { debounce } from 'throttle-debounce';

import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import EditableTagList from 'components/TagList';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import useModal, { ModalHooks as Hooks } from 'hooks/useModal/useModal';
import usePrevious from 'hooks/usePrevious';
import { paths } from 'routes/utils';
import { getModels, postModel } from 'services/api';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { Metadata } from 'types';
import handleError from 'utils/error';

import css from './useModalModelCreate.module.scss';

interface OpenProps {
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

interface ModalHooks extends Omit<Hooks, 'modalOpen'> {
  modalOpen: (openProps?: OpenProps) => void;
}

const DEFAULT_MODAL_STATE = {
  expandDetails: false,
  isNameUnique: true,
  metadata: {},
  modelDescription: '',
  modelName: '',
  tags: [],
  visible: false,
};

const useModalModelCreate = (): ModalHooks => {
  const [ modalState, setModalState ] = useState<ModalState>(DEFAULT_MODAL_STATE);
  const prevModalState = usePrevious(modalState, DEFAULT_MODAL_STATE);

  const { modalOpen: showRegisterCheckpointModal } = useModalCheckpointRegister();

  const { modalClose, modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();

  const modalOpen = useCallback(({ checkpointUuid }: OpenProps = {}) => {
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
      modalClose();

      if (checkpointUuid) {
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
  }, [ modalClose, showRegisterCheckpointModal ]);

  const handleOk = useCallback(async (state: ModalState) => {
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

  const getModalContent = useCallback((state: ModalState): React.ReactNode => {
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
          {!isNameUnique && (
            <p className={css.uniqueWarning}>A model with this name already exists</p>
          )}
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
  }, [
    openDetails,
    updateMetadata,
    updateTags,
    updateModelDescription,
    updateModelName,
  ]);

  const handleCancel = useCallback(() => modalClose(), [ modalClose ]);

  const getModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    const { modelName, isNameUnique } = state;
    return {
      bodyStyle: { padding: 0 },
      className: css.base,
      closable: true,
      content: getModalContent(state),
      icon: null,
      maskClosable: true,
      okButtonProps: { disabled: modelName === '' || !isNameUnique },
      okText: 'Create Model',
      onCancel: handleCancel,
      onOk: () => handleOk(state),
      title: 'Create Model',
    };
  }, [ getModalContent, handleCancel, handleOk ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (isEqual(modalState, prevModalState) || !modalState.visible) return;
    openOrUpdate(getModalProps(modalState));
  }, [ getModalProps, modalState, openOrUpdate, prevModalState ]);

  return { modalClose, modalOpen, modalRef, ...modalHook };
};

export default useModalModelCreate;
