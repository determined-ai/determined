import { Divider, Form, Input, InputNumber, Switch } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { createWorkspace, getWorkspace, patchWorkspace } from 'services/api';
import Spinner from 'shared/components/Spinner';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { Workspace } from 'types';
import handleError from 'utils/error';

const FORM_ID = 'new-workspace-form';

interface FormInputs {
  agentGid?: number;
  agentGroup?: string;
  agentUid?: number;
  agentUser?: string;
  checkpointStorageConfig?: string;
  useAgentGroup: boolean;
  useAgentUser: boolean;
  useCheckpointStorage: boolean;
  workspaceName: string;
}

interface Props {
  onClose?: () => void;
  workspaceID?: number;
}

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const useModalWorkspaceCreate = ({ onClose, workspaceID }: Props = {}): ModalHooks => {
  const { canModifyWorkspaceAgentUserGroup, canModifyWorkspaceCheckpointStorage } =
    usePermissions();
  const [form] = Form.useForm<FormInputs>();
  const workspaceName = Form.useWatch('workspaceName', form);
  const useAgentUser = Form.useWatch('useAgentUser', form);
  const useAgentGroup = Form.useWatch('useAgentGroup', form);
  const useCheckpointStorage = Form.useWatch('useCheckpointStorage', form);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const [canceler] = useState(new AbortController());
  const [workspace, setWorkspace] = useState<Workspace>();

  const fetchWorkspace = useCallback(async () => {
    if (workspaceID) {
      try {
        const response = await getWorkspace({ id: workspaceID }, { signal: canceler.signal });
        setWorkspace(response);
      } catch (e) {
        handleError(e);
      }
    }
  }, [workspaceID, canceler.signal]);

  useEffect(() => {
    if (workspace) {
      form.resetFields();
      const { name, checkpointStorageConfig, agentUserGroup } = workspace;
      form.setFieldsValue({
        checkpointStorageConfig: yaml.dump(checkpointStorageConfig),
        useCheckpointStorage: !!checkpointStorageConfig,
        workspaceName: name,
      });
      if (agentUserGroup) {
        const { agentUid, agentGid, agentGroup, agentUser } = agentUserGroup;
        form.setFieldsValue({
          agentGid,
          agentGroup,
          agentUid,
          agentUser,
          useAgentGroup: !!agentGid && !!agentGroup,
          useAgentUser: !!agentUid && !!agentUser,
        });
      }
    }
  }, [workspace, form]);

  const modalContent = useMemo(() => {
    if (workspaceID && !workspace) return <Spinner />;
    return (
      <Form autoComplete="off" form={form} id={FORM_ID} labelCol={{ span: 10 }} layout="vertical">
        <Form.Item
          label="Workspace Name"
          name="workspaceName"
          rules={[{ message: 'Workspace name is required ', required: true }]}>
          <Input maxLength={80} />
        </Form.Item>
        {canModifyWorkspaceAgentUserGroup({ workspace }) && (
          <>
            <Divider />
            <Form.Item label="Configure Agent User" name="useAgentUser" valuePropName="checked">
              <Switch />
            </Form.Item>
            {useAgentUser && (
              <>
                <Form.Item
                  label="Agent User ID"
                  name="agentUid"
                  rules={[{ message: 'Agent User ID is required ', required: true }]}>
                  <InputNumber disabled={!canModifyWorkspaceAgentUserGroup({ workspace })} />
                </Form.Item>
                <Form.Item
                  label="Agent User Name"
                  name="agentUser"
                  rules={[{ message: 'Agent User Name is required ', required: true }]}>
                  <Input
                    disabled={!canModifyWorkspaceAgentUserGroup({ workspace })}
                    maxLength={100}
                  />
                </Form.Item>
              </>
            )}
            <Form.Item label="Configure Agent Group" name="useAgentGroup" valuePropName="checked">
              <Switch />
            </Form.Item>
            {useAgentGroup && (
              <>
                <Form.Item
                  label="Agent User Group ID"
                  name="agentGid"
                  rules={[{ message: 'Agent User Group ID is required ', required: true }]}>
                  <InputNumber disabled={!canModifyWorkspaceAgentUserGroup({ workspace })} />
                </Form.Item>
                <Form.Item
                  label="Agent Group Name"
                  name="agentGroup"
                  rules={[{ message: 'Agent Group Name is required ', required: true }]}>
                  <Input
                    disabled={!canModifyWorkspaceAgentUserGroup({ workspace })}
                    maxLength={100}
                  />
                </Form.Item>
              </>
            )}
          </>
        )}
        {canModifyWorkspaceCheckpointStorage({ workspace }) && (
          <>
            <Divider />
            <Form.Item
              label="Configure Checkpoint Storage"
              name="useCheckpointStorage"
              valuePropName="checked">
              <Switch />
            </Form.Item>
            {useCheckpointStorage && (
              <React.Suspense fallback={<Spinner tip="Loading text editor..." />}>
                <Form.Item
                  label="Checkpoint Storage"
                  name="checkpointStorageConfig"
                  rules={[
                    { message: 'Checkpoint Storage config is required', required: true },
                    {
                      validator: (_, value) => {
                        try {
                          yaml.load(value);
                          return Promise.resolve();
                        } catch (err: unknown) {
                          return Promise.reject(
                            new Error(
                              `Invalid YAML on line ${
                                (err as { mark: { line: string } }).mark.line
                              }.`,
                            ),
                          );
                        }
                      },
                    },
                  ]}>
                  <MonacoEditor
                    height="16vh"
                    options={{
                      readOnly: !canModifyWorkspaceCheckpointStorage({ workspace }),
                      wordWrap: 'on',
                      wrappingIndent: 'indent',
                    }}
                  />
                </Form.Item>
              </React.Suspense>
            )}
          </>
        )}
      </Form>
    );
  }, [
    form,
    useAgentUser,
    useAgentGroup,
    useCheckpointStorage,
    workspace,
    workspaceID,
    canModifyWorkspaceAgentUserGroup,
    canModifyWorkspaceCheckpointStorage,
  ]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        const {
          workspaceName,
          agentUid,
          agentUser,
          agentGid,
          agentGroup,
          useAgentUser,
          useAgentGroup,
          checkpointStorageConfig,
        } = values;
        const body = {
          name: workspaceName,
        };

        if (canModifyWorkspaceAgentUserGroup({ workspace })) {
          let agentUserGroup = {};
          useAgentUser && (agentUserGroup = { agentUid, agentUser });
          useAgentGroup && (agentUserGroup = { agentGid, agentGroup, ...agentUserGroup });
          body['agentUserGroup'] = agentUserGroup;
        }

        if (canModifyWorkspaceCheckpointStorage({ workspace })) {
          if (useCheckpointStorage && checkpointStorageConfig) {
            body['checkpointStorageConfig'] = yaml.load(checkpointStorageConfig);
          } else {
            body['checkpointStorageConfig'] = {};
          }
        }

        if (workspaceID) {
          const response = await patchWorkspace({ id: workspaceID, ...body });
          setWorkspace(response);
        } else {
          const response = await createWorkspace(body);
          routeToReactUrl(paths.workspaceDetails(response.id));
        }
        form.resetFields();
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to save workspace.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create workspace.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [
    form,
    workspaceID,
    workspace,
    canModifyWorkspaceCheckpointStorage,
    canModifyWorkspaceAgentUserGroup,
  ]);

  const getModalProps = useMemo((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: !workspaceName, form: FORM_ID, htmlType: 'submit' },
      okText: 'Save Workspace',
      onOk: handleOk,
      title: `${workspaceID ? 'Edit' : 'New'} Workspace`,
      width: 600,
    };
  }, [handleOk, modalContent, workspaceName, workspaceID]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...getModalProps, ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps);
  }, [getModalProps, modalRef, openOrUpdate]);

  useEffect(() => {
    modalRef.current && workspaceID !== workspace?.id && fetchWorkspace();
  });

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceCreate;
