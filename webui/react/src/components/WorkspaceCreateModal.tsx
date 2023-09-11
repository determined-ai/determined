import { Divider, Switch } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import { Modal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { patchWorkspace } from 'services/api';
import { V1AgentUserGroup } from 'services/api-ts-sdk';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { routeToReactUrl } from 'utils/routes';

const FORM_ID = 'create-workspace-form';

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
  workspaceId?: number;
}

const CodeEditor = React.lazy(() => import('components/kit/CodeEditor'));

const WorkspaceCreateModalComponent: React.FC<Props> = ({ onClose, workspaceId }: Props = {}) => {
  const { canModifyWorkspaceAgentUserGroup, canModifyWorkspaceCheckpointStorage } =
    usePermissions();
  const [form] = Form.useForm<FormInputs>();
  const useAgentUser = Form.useWatch('useAgentUser', form);
  const useAgentGroup = Form.useWatch('useAgentGroup', form);
  const useCheckpointStorage = Form.useWatch('useCheckpointStorage', form);

  const initFields = useCallback(
    (ws?: Workspace) => {
      if (ws) {
        form.resetFields();
        const { name, checkpointStorageConfig, agentUserGroup } = ws;
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
    },
    [form],
  );

  const loadableWorkspace = useObservable(workspaceStore.getWorkspace(workspaceId || 0));
  const workspace = Loadable.getOrElse(undefined, loadableWorkspace);
  useEffect(() => {
    initFields(workspace || undefined);
  }, [workspace, initFields]);

  const [canModifyAUG, canModifyCPS] = useMemo(() => {
    const workspace = workspaceId ? { id: workspaceId } : undefined;
    return [
      canModifyWorkspaceAgentUserGroup({ workspace }),
      canModifyWorkspaceCheckpointStorage({ workspace }),
    ];
  }, [canModifyWorkspaceAgentUserGroup, canModifyWorkspaceCheckpointStorage, workspaceId]);

  const modalContent = useMemo(() => {
    if (workspaceId && loadableWorkspace === NotLoaded) return <Spinner spinning />;
    return (
      <Form autoComplete="off" form={form} id={FORM_ID} labelCol={{ span: 10 }} layout="vertical">
        <Form.Item
          label="Workspace Name"
          name="workspaceName"
          rules={[
            {
              message: 'Name must be 1 ~ 80 letters, and contain at least non-whitespace letter',
              pattern: new RegExp('.*[^ ].*'),
              required: true,
            },
          ]}>
          <Input maxLength={80} />
        </Form.Item>
        {canModifyAUG && (
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
                  <InputNumber disabled={!canModifyAUG} />
                </Form.Item>
                <Form.Item
                  label="Agent User Name"
                  name="agentUser"
                  rules={[{ message: 'Agent User Name is required ', required: true }]}>
                  <Input disabled={!canModifyAUG} maxLength={100} />
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
                  <InputNumber disabled={!canModifyAUG} />
                </Form.Item>
                <Form.Item
                  label="Agent Group Name"
                  name="agentGroup"
                  rules={[{ message: 'Agent Group Name is required ', required: true }]}>
                  <Input disabled={!canModifyAUG} maxLength={100} />
                </Form.Item>
              </>
            )}
          </>
        )}
        {canModifyCPS && (
          <>
            <Divider />
            <Form.Item
              label="Configure Checkpoint Storage"
              name="useCheckpointStorage"
              valuePropName="checked">
              <Switch />
            </Form.Item>
            {useCheckpointStorage && (
              <React.Suspense fallback={<Spinner spinning tip="Loading text editor..." />}>
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
                  <CodeEditor
                    file={Loaded('')}
                    files={[{ key: 'config.yaml' }]}
                    height="16vh"
                    readonly={!canModifyCPS}
                    onError={handleError}
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
    loadableWorkspace,
    workspaceId,
    canModifyAUG,
    canModifyCPS,
  ]);

  const handleSubmit = useCallback(async () => {
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
        const body: {
          agentUserGroup?: V1AgentUserGroup;
          checkpointStorageConfig?: unknown;
          name: string;
        } = {
          name: workspaceName,
        };

        if (canModifyAUG) {
          let agentUserGroup = {};
          useAgentUser && (agentUserGroup = { agentUid, agentUser });
          useAgentGroup && (agentUserGroup = { agentGid, agentGroup, ...agentUserGroup });
          body['agentUserGroup'] = agentUserGroup;
        }

        if (canModifyCPS) {
          if (checkpointStorageConfig) {
            body['checkpointStorageConfig'] = yaml.load(checkpointStorageConfig);
          } else {
            body['checkpointStorageConfig'] = {};
          }
        }

        if (workspaceId) {
          await patchWorkspace({ id: workspaceId, ...body });
          workspaceStore.fetch(undefined, true);
        } else {
          const response = await workspaceStore.createWorkspace(body);
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
  }, [form, workspaceId, canModifyAUG, canModifyCPS]);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        form: FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Save Workspace',
      }}
      title={`${workspaceId ? 'Edit' : 'New'} Workspace`}
      onClose={() => {
        initFields(undefined);
        onClose?.();
      }}>
      {modalContent}
    </Modal>
  );
};

export default WorkspaceCreateModalComponent;
