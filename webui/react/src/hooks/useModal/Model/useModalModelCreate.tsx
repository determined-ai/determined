import { ModalFuncProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import Tags, { tagsActionHelper } from 'components/kit/Tags';
import Tooltip from 'components/kit/Tooltip';
import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import SelectFilter from 'components/SelectFilter';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { postModel } from 'services/api';
import useModal, { ModalHooks as Hooks, ModalCloseReason } from 'shared/hooks/useModal/useModal';
import usePrevious from 'shared/hooks/usePrevious';
import { clone, isEqual } from 'shared/utils/data';
import { DetError, ErrorType } from 'shared/utils/error';
import { useWorkspaces } from 'stores/workspaces';
import { Metadata } from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

import css from './useModalModelCreate.module.scss';

const FORM_ID = 'create-model-form';

interface Props {
  onClose?: (reason?: ModalCloseReason, checkpoints?: string[], modelName?: string) => void;
  workspaceId?: number;
}

interface FormInputs {
  description?: string;
  modelName: string;
  workspaceId: number;
}

interface OpenProps {
  checkpoints?: string[];
}

interface ModalState {
  checkpoints?: string[];
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

const useModalModelCreate = ({ onClose, workspaceId }: Props = {}): ModalHooks => {
  const { canCreateModelWorkspace } = usePermissions();
  const loadableWorkspaces = useWorkspaces();
  const isWorkspace = workspaceId !== undefined;
  const workspaces = Loadable.match(loadableWorkspaces, {
    Loaded: (ws) => ws.filter(({ id }) => canCreateModelWorkspace({ workspaceId: id })),
    NotLoaded: () => [],
  });
  const [modalState, setModalState] = useState<ModalState>(DEFAULT_MODAL_STATE);
  const prevModalState = usePrevious(modalState, undefined);
  const [form] = Form.useForm<FormInputs>();
  const modelName = Form.useWatch('modelName', form);
  const handleOnClose = useCallback(
    (reason?: ModalCloseReason) => {
      onClose?.(reason, modalState.checkpoints, modalState.modelName || undefined);
      setModalState(DEFAULT_MODAL_STATE);
    },
    [modalState, onClose],
  );

  const {
    modalClose,
    modalOpen: openOrUpdate,
    ...modalHook
  } = useModal({ onClose: handleOnClose });

  const modalOpen = useCallback(({ checkpoints }: OpenProps = {}) => {
    const newState = clone(DEFAULT_MODAL_STATE);
    setModalState({ ...newState, checkpoints, visible: true });
  }, []);

  const createModel = useCallback(async (state: ModalState, workspaceId: number) => {
    const { modelDescription, tags, metadata, modelName } = state;
    try {
      const response = await postModel({
        description: modelDescription,
        labels: tags,
        metadata: metadata,
        name: modelName,
        workspaceId,
      });
      if (!response?.id) return;

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
  }, []);

  const handleOk = useCallback(
    async (state: ModalState) => {
      const values = await form.validateFields();
      if (values) {
        await createModel(
          {
            ...state,
            modelDescription: values.description ?? '',
            modelName: values.modelName,
          },
          values.workspaceId,
        );
        form.resetFields();
      }
    },
    [createModel, form],
  );

  const openDetails = useCallback(() => {
    setModalState((prev) => ({ ...prev, expandDetails: true }));
  }, []);

  const handleMetadataChange = useCallback((value: Metadata) => {
    setModalState((prev) => ({ ...prev, metadata: value }));
  }, []);

  const handleTagsChange = useCallback((value: string[]) => {
    setModalState((prev) => ({ ...prev, tags: value }));
  }, []);

  const handleNameChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setModalState((prev) => ({ ...prev, modelName: e.target.value }));
  }, []);

  const handleDescriptionChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setModalState((prev) => ({ ...prev, modelDescription: e.target.value }));
  }, []);

  const workspaceItems = useMemo(
    () =>
      workspaces.map((ws) => ({
        label: ws.name,
        value: ws.id,
      })),
    [workspaces],
  );

  const getModalContent = useCallback(
    (state: ModalState): React.ReactNode => {
      const { tags, metadata, expandDetails } = state;

      // We always render the form regardless of mode to provide a reference to it.
      return (
        <>
          <Form autoComplete="off" form={form} id={FORM_ID} layout="vertical">
            <p className={css.directions}>
              Create a registered model to organize important checkpoints.
            </p>
            <Form.Item
              initialValue={workspaceId}
              label="Workspace"
              name="workspaceId"
              rules={[{ message: 'Please select a workspace', required: true }]}>
              <SelectFilter
                disabled={!workspaces.length || isWorkspace}
                filterOption={(input, option) =>
                  (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
                }
                options={workspaceItems}
                placeholder="Select a workspace"
                showSearch={true}
              />
            </Form.Item>
            <Form.Item
              label="Model name"
              name="modelName"
              rules={[{ message: 'Please input Model name', required: true }]}>
              <Input onChange={handleNameChange} />
            </Form.Item>
            <Form.Item label="Description (optional)" name="description">
              <Input.TextArea onChange={handleDescriptionChange} />
            </Form.Item>
            {!expandDetails && (
              <p className={css.expandDetails} onClick={openDetails}>
                Add More Details...
              </p>
            )}
          </Form>
          {expandDetails && (
            <>
              <div>
                <h2>
                  Metadata <span>(optional)</span>
                </h2>
                <EditableMetadata
                  editing={true}
                  metadata={metadata}
                  updateMetadata={handleMetadataChange}
                />
              </div>
              <div>
                <h2>
                  Tags <span>(optional)</span>
                </h2>
                <Tags tags={tags} onAction={tagsActionHelper(tags, handleTagsChange)} />
              </div>
            </>
          )}
        </>
      );
    },
    [
      form,
      workspaceId,
      workspaces.length,
      isWorkspace,
      workspaceItems,
      handleDescriptionChange,
      handleMetadataChange,
      handleNameChange,
      handleTagsChange,
      openDetails,
    ],
  );

  const getModalProps = useCallback(
    (state: ModalState): Partial<ModalFuncProps> => {
      const disableWorkspaceModelCreation = isWorkspace
        ? !canCreateModelWorkspace({ workspaceId })
        : false;
      return {
        className: css.base,
        closable: true,
        content: getModalContent(state),
        icon: null,
        maskClosable: true,
        okButtonProps: {
          disabled: !modelName || (isWorkspace && disableWorkspaceModelCreation),
          form: FORM_ID,
          htmlType: 'submit',
        },
        okText: (
          <Tooltip
            placement="bottom"
            title={
              disableWorkspaceModelCreation
                ? 'You do not have permission to create a model in this workspace!'
                : undefined
            }>
            Create Model
          </Tooltip>
        ),
        onOk: () => handleOk(state),
        title: '',
      };
    },
    [getModalContent, handleOk, modelName, isWorkspace, workspaceId, canCreateModelWorkspace],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (isEqual(modalState, prevModalState) || !modalState.visible) return;
    openOrUpdate(getModalProps(modalState));
  }, [getModalProps, modalState, openOrUpdate, prevModalState]);

  return { modalClose, modalOpen, ...modalHook };
};

export default useModalModelCreate;
