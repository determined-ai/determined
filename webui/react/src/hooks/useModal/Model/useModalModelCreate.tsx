import { DownOutlined } from '@ant-design/icons';
import { Button, Dropdown, MenuProps, ModalFuncProps, Tooltip } from 'antd';
import { SelectInfo } from 'rc-menu/lib/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';

import DynamicIcon from 'components/DynamicIcon';
import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import EditableTagList from 'components/TagList';
import useFeature from 'hooks/useFeature';
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
}

interface FormInputs {
  description?: string;
  modelName: string;
  workspace: string;
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
  workspace: string;
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
  workspace: '1',
};

const useModalModelCreate = ({ onClose }: Props = {}): ModalHooks => {
  const { canCreateModelWorkspace } = usePermissions();
  const isRbacEnabled = useFeature().isOn('model_rbac');
  const loadableWorkspaces = useWorkspaces();
  const location = useLocation().pathname;
  const isWorkspace = location.includes('/workspaces');
  const workspaceId = useMemo(
    () => (isWorkspace ? location.split('/')[2] : ''),
    [isWorkspace, location],
  );
  const workspaces = Loadable.match(loadableWorkspaces, {
    Loaded: (ws) => ws.filter(({ id }) => canCreateModelWorkspace(id)),
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

  useEffect(() => {
    if (workspaceId) setModalState({ ...modalState, workspace: workspaceId });
  }, [workspaceId, modalState]);

  const {
    modalClose,
    modalOpen: openOrUpdate,
    ...modalHook
  } = useModal({ onClose: handleOnClose });

  const modalOpen = useCallback(({ checkpoints }: OpenProps = {}) => {
    const newState = clone(DEFAULT_MODAL_STATE);
    setModalState({ ...newState, checkpoints, visible: true });
  }, []);

  const createModel = useCallback(async (state: ModalState) => {
    const { modelDescription, tags, metadata, modelName } = state;
    try {
      const response = await postModel({
        description: modelDescription,
        labels: tags,
        metadata: metadata,
        name: modelName,
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
        await createModel({
          ...state,
          modelDescription: values.description ?? '',
          modelName: values.modelName,
        });
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

  const onSelect = useCallback((info: SelectInfo) => {
    setModalState((prev) => ({ ...prev, workspace: info.key }));
  }, []);

  const workspaceName = useMemo(() => {
    const name = workspaces.find((ws) => String(ws.id) === modalState.workspace)?.name || '';

    return name;
  }, [workspaces, modalState.workspace]);

  const workspaceItems: MenuProps['items'] = useMemo(
    () =>
      workspaces.map((ws) => ({
        key: ws.id,
        label: (
          <div className={css.workspaceFilterItem}>
            <DynamicIcon name={ws.name} size={24} />
            <span className={css.workspaceFilterName}>{ws.name}</span>
          </div>
        ),
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
              label="Model name"
              name="modelName"
              required
              requiredMessage="Model name is required ">
              <Input onChange={handleNameChange} />
            </Form.Item>
            {isRbacEnabled && (
              <Tooltip placement="bottom">
                <Form.Item label="Workspaces" name="workspace">
                  <Dropdown
                    arrow
                    className={css.workspacDropdown}
                    disabled={!workspaces.length || isWorkspace}
                    dropdownRender={(menu) =>
                      React.cloneElement(menu as React.ReactElement, {
                        style: { maxHeight: '200px', maxWidth: 'fit-content', overflowY: 'scroll' },
                      })
                    }
                    getPopupContainer={(triggerNode) => triggerNode}
                    menu={{
                      defaultSelectedKeys: [modalState.workspace],
                      items: workspaceItems,
                      onSelect,
                      selectable: true,
                    }}
                    trigger={['click']}>
                    <Button>
                      <span
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          minWidth: '175px',
                        }}>
                        {workspaceName}
                        <DownOutlined style={{ marginLeft: '10px' }} />
                      </span>
                    </Button>
                  </Dropdown>
                </Form.Item>
              </Tooltip>
            )}
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
                <EditableTagList tags={tags} onChange={handleTagsChange} />
              </div>
            </>
          )}
        </>
      );
    },
    [
      form,
      isWorkspace,
      workspaceItems,
      workspaces,
      workspaceName,
      modalState.workspace,
      onSelect,
      handleDescriptionChange,
      handleMetadataChange,
      handleNameChange,
      handleTagsChange,
      openDetails,
    ],
  );

  const getModalProps = useCallback(
    (state: ModalState): Partial<ModalFuncProps> => {
      return {
        className: css.base,
        closable: true,
        content: getModalContent(state),
        icon: null,
        maskClosable: true,
        okButtonProps: {
          disabled: !modelName,
          form: FORM_ID,
          htmlType: 'submit',
        },
        okText: 'Create Model',
        onOk: () => handleOk(state),
        title: 'Create Model',
      };
    },
    [getModalContent, handleOk, modelName],
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
