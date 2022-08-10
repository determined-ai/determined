import { Form, Input, ModalFuncProps, notification } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import EditableTagList from 'components/TagList';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import { paths } from 'routes/utils';
import { postModel } from 'services/api';
import useModal, { ModalHooks as Hooks } from 'shared/hooks/useModal/useModal';
import usePrevious from 'shared/hooks/usePrevious';
import { clone, isEqual } from 'shared/utils/data';
import { DetError, ErrorType } from 'shared/utils/error';
import { Metadata } from 'types';
import handleError from 'utils/error';

import css from './useModalModelCreate.module.scss';

interface FormInputs {
  description?: string;
  modelName: string;
}

interface OpenProps {
  checkpointUuid?: string;
}

interface ModalState {
  checkpointUuid?: string;
  expandDetails: boolean;
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
  metadata: {},
  modelDescription: '',
  modelName: '',
  tags: [],
  visible: false,
};

const useModalModelCreate = (): ModalHooks => {
  const [ modalState, setModalState ] = useState<ModalState>(DEFAULT_MODAL_STATE);
  const prevModalState = usePrevious(modalState, undefined);
  const [ form ] = Form.useForm<FormInputs>();

  const handleOnCancel = useCallback(() => {
    setModalState(DEFAULT_MODAL_STATE);
  }, []);

  const { modalOpen: modalOpenCheckpointRegister } = useModalCheckpointRegister(
    { onClose: handleOnCancel },
  );

  const handleOnClose = useCallback(() => {
    setModalState(DEFAULT_MODAL_STATE);
  }, []);

  const {
    modalClose,
    modalOpen: openOrUpdate,
    ...modalHook
  } = useModal({ onClose: handleOnClose });

  const modalOpen = useCallback(({ checkpointUuid }: OpenProps = {}) => {
    const newState = clone(DEFAULT_MODAL_STATE);
    setModalState({ ...newState, checkpointUuid, visible: true });
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

      if (checkpointUuid) {
        modalOpenCheckpointRegister({ checkpointUuid, selectedModelName: response.name });
      }

      notification.open({
        btn: null,
        description: (
          <div className={css.toast}>
            <p>{`"${modelName}"`} created</p>
            <Link path={paths.modelDetails(response.name)}>View Model</Link>
          </div>
        ),
        message: '',
      });
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create model.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create model.',
          silent: false,
          type: ErrorType.Api,
        });
      }
    }
  }, [ modalOpenCheckpointRegister ]);

  const handleOk = useCallback(async (state: ModalState) => {
    const values = await form.validateFields();
    if (values) {
      await createModel({
        ...state,
        modelDescription: values.description ?? '',
        modelName: values.modelName,
      });
      form.resetFields();
    }
  }, [ createModel, form ]);

  const openDetails = useCallback(() => {
    setModalState((prev) => ({ ...prev, expandDetails: true }));
  }, []);

  const updateMetadata = useCallback((value) => {
    setModalState((prev) => ({ ...prev, metadata: value }));
  }, []);

  const updateTags = useCallback((value) => {
    setModalState((prev) => ({ ...prev, tags: value }));
  }, []);

  const getModalContent = useCallback((state: ModalState): React.ReactNode => {
    const { tags, metadata, expandDetails } = state;

    // We always render the form regardless of mode to provide a reference to it.
    return (
      <Form autoComplete="off" form={form} layout="vertical">
        <p className={css.directions}>
          Create a registered model to organize important checkpoints.
        </p>
        <Form.Item
          label="Model name"
          name="modelName"
          rules={[ { message: 'Model name is required ', required: true } ]}>
          <Input />
        </Form.Item>
        <Form.Item label="Description (optional)" name="description">
          <Input.TextArea />
        </Form.Item>
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
      </Form>
    );
  }, [ form, updateMetadata, updateTags, openDetails ]);

  const handleCancel = useCallback(() => modalClose(), [ modalClose ]);

  const getModalProps = useCallback((state: ModalState): Partial<ModalFuncProps> => {
    return {
      className: css.base,
      closable: true,
      content: getModalContent(state),
      icon: null,
      maskClosable: true,
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

  return { modalClose, modalOpen, ...modalHook };
};

export default useModalModelCreate;
