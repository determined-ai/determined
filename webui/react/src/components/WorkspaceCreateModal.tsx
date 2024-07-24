import Divider from 'hew/Divider';
import Form from 'hew/Form';
import Input from 'hew/Input';
import InputNumber from 'hew/InputNumber';
import { Modal } from 'hew/Modal';
import Spinner from 'hew/Spinner';
import Toggle from 'hew/Toggle';
import { Body } from 'hew/Typography';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import yaml from 'js-yaml';
import { pick } from 'lodash';
import React, { Fragment, useCallback, useEffect, useId, useMemo } from 'react';

import { useAsync } from 'hooks/useAsync';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import {
  getKubernetesResourceManagers,
  getKubernetesResourceQuotas,
  listWorkspaceNamespaceBindings,
  patchWorkspace,
  setResourceQuotas,
} from 'services/api';
import {
  V1AgentUserGroup,
  V1WorkspaceNamespaceBinding,
  V1WorkspaceNamespaceMeta,
} from 'services/api-ts-sdk';
import determinedStore, { BrandingType } from 'stores/determinedInfo';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
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
  bindings?: Record<string, V1WorkspaceNamespaceBinding>;
  resourceQuotas?: Record<string, number>;
}

interface Props {
  onClose?: () => void;
  workspaceId?: number;
}

const CodeEditor = React.lazy(() => import('hew/CodeEditor'));

const isNonK8RMError = (e: unknown): boolean => {
  return e instanceof DetError && e.sourceErr instanceof Response && e.sourceErr['status'] === 404;
};

const isNotAuthorizedErr = (e: unknown): boolean => {
  return e instanceof DetError && e.sourceErr instanceof Response && (e.sourceErr['status'] === 403 || e.sourceErr['status'] === 401);
};

const WorkspaceCreateModalComponent: React.FC<Props> = ({ onClose, workspaceId }: Props = {}) => {
  const idPrefix = useId();
  const {
    canModifyWorkspaceAgentUserGroup,
    canModifyWorkspaceCheckpointStorage,
    canSetWorkspaceNamespaceBindings,
  } = usePermissions();
  const info = useObservable(determinedStore.info);
  const [form] = Form.useForm<FormInputs>();
  const useAgentUser = Form.useWatch('useAgentUser', form);
  const useAgentGroup = Form.useWatch('useAgentGroup', form);
  const useCheckpointStorage = Form.useWatch('useCheckpointStorage', form);
  const watchBindings = Form.useWatch('bindings', form);
  const resourceManagers = useAsync(async (canceller) => {
    try {
      const response = await getKubernetesResourceManagers(undefined, { signal: canceller.signal });
      return response.names;
    } catch (e) {
      if (!isNotAuthorizedErr(e)) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Failed to fetch Resource Managers.',
          silent: false,
          type: ErrorType.Server,
        });
      }
      return NotLoaded;
    }
  }, []);

  const namespaceBindingsList = useAsync(
    async (canceller) => {
      if (workspaceId === undefined) {
        return NotLoaded;
      }
      try {
        const clusterNamespacePairs = await listWorkspaceNamespaceBindings(
          { id: workspaceId },
          { signal: canceller.signal },
        );
        return clusterNamespacePairs.namespaceBindings;
      } catch (e) {
        if (!isNonK8RMError(e)) {
          handleError(e, {
            level: ErrorLevel.Error,
            publicMessage: 'Failed to fetch list of workspace namespace bindings.',
            silent: true,
            type: ErrorType.Server,
          });
        }
        return NotLoaded;
      }
    },
    [workspaceId],
  );

  const resourceQuotasList = useAsync(
    async (canceller) => {
      if (workspaceId === undefined) {
        return NotLoaded;
      }
      try {
        const resp = await getKubernetesResourceQuotas(
          { id: workspaceId },
          { signal: canceller.signal },
        );
        return resp.resourceQuotas;
      } catch (e) {
        if (!isNonK8RMError(e)) {
          handleError(e, {
            level: ErrorLevel.Error,
            publicMessage: 'Failed to fetch kubernetes resource quotas for the workspace.',
            silent: false,
            type: ErrorType.Server,
          });
        }
        return NotLoaded;
      }
    },
    [workspaceId],
  );

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
        namespaceBindingsList.forEach((bindingsList) => {
          form.setFieldValue('bindings', { ...bindingsList });
        });
        resourceQuotasList.forEach((quotasList) => {
          form.setFieldValue('resourceQuotas', { ...quotasList });
        });
      }
    },
    [form, namespaceBindingsList, resourceQuotasList],
  );

  const loadableWorkspace = useObservable(workspaceStore.getWorkspace(workspaceId || 0));
  const workspace = Loadable.getOrElse(undefined, loadableWorkspace);

  useEffect(() => {
    initFields(workspace || undefined);
  }, [workspace, initFields]);

  const [canModifyAUG, canModifyCPS, canModifyBindings] = useMemo(() => {
    const workspace = workspaceId ? { id: workspaceId } : undefined;
    return [
      canModifyWorkspaceAgentUserGroup({ workspace }),
      canModifyWorkspaceCheckpointStorage({ workspace }),
      canSetWorkspaceNamespaceBindings({ workspace }),
    ];
  }, [
    canModifyWorkspaceAgentUserGroup,
    canModifyWorkspaceCheckpointStorage,
    canSetWorkspaceNamespaceBindings,
    workspaceId,
  ]);

  const modalContent = useMemo(() => {
    if (workspaceId && loadableWorkspace === NotLoaded) return <Spinner spinning />;
    return (
      <Form
        autoComplete="off"
        form={form}
        id={idPrefix + FORM_ID}
        labelCol={{ span: 10 }}
        layout="vertical">
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
        {canModifyBindings && Loadable.getOrElse([], resourceManagers).length > 0 && (
          <>
            <Divider />
            Namespace Bindings
            <Body inactive>
              Note: If you leave the Namespace name blank, the workspace will be bound to the
              default Namespace configured in the Master Config.
            </Body>
            <>
              {Loadable.getOrElse([], resourceManagers).map((name) => (
                <Fragment key={name}>
                  <Form.Item label={name} name={['bindings', name, 'namespace']}>
                    <Input
                      disabled={watchBindings?.[name]?.['autoCreateNamespace'] ?? false}
                      maxLength={63}
                    />
                  </Form.Item>
                  {info.branding === BrandingType.HPE && (
                    <>
                      <Form.Item
                        label="Auto Create Namespace"
                        name={['bindings', name, 'autoCreateNamespace']}
                        valuePropName="checked">
                        <Toggle
                          onChange={() => form.setFieldValue(['resourceQuotas', name], undefined)}
                        />
                      </Form.Item>
                      <Form.Item
                        label="Resource Quota"
                        name={['resourceQuotas', name]}
                        rules={[
                          {
                            message: 'Resource Quota has to be greater or equal to 0',
                            min: 0,
                            type: 'number',
                          },
                        ]}>
                        <InputNumber
                          disabled={!(watchBindings?.[name]?.['autoCreateNamespace'] ?? false)}
                          min={0}
                        />
                      </Form.Item>
                    </>
                  )}
                </Fragment>
              ))}
            </>
          </>
        )}
        {canModifyAUG && (
          <>
            <Divider />
            <Form.Item
              data-testid="useAgentUser"
              label="Configure Agent User"
              name="useAgentUser"
              valuePropName="checked">
              <Toggle />
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
            <Form.Item
              data-testid="useAgentGroup"
              label="Configure Agent Group"
              name="useAgentGroup"
              valuePropName="checked">
              <Toggle />
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
              data-testid="useCheckpointStorage"
              label="Configure Checkpoint Storage"
              name="useCheckpointStorage"
              valuePropName="checked">
              <Toggle />
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
    workspaceId,
    loadableWorkspace,
    form,
    idPrefix,
    info.branding,
    canModifyBindings,
    resourceManagers,
    canModifyAUG,
    useAgentUser,
    useAgentGroup,
    canModifyCPS,
    useCheckpointStorage,
    watchBindings,
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
          clusterNamespaceMeta?: Record<string, V1WorkspaceNamespaceMeta>;
        } = {
          name: workspaceName,
        };
        const resourceQuotaBody: {
          id: number;
          clusterQuotaPairs?: Record<string, number>;
        } = {
          id: 0,
        };
        const clusterNamespaceMeta = Object.keys(values.bindings ?? {}).reduce(
          (memo: Record<string, V1WorkspaceNamespaceMeta>, name) => {
            if (values.bindings !== undefined) {
              const data = values.bindings[name];
              if (data.autoCreateNamespace) {
                data.namespace = undefined;
                memo[name] = data;
              } else {
                if (values.resourceQuotas?.[name]) {
                  delete values.resourceQuotas[name];
                }
                if (data.namespace !== undefined && data.namespace !== '') {
                  data.autoCreateNamespace = false;
                  memo[name] = data;
                }
              }
            }
            return memo;
          },
          {},
        );
        body['clusterNamespaceMeta'] = clusterNamespaceMeta;

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
          resourceQuotaBody['id'] = workspaceId;
        } else {
          const response = await workspaceStore.createWorkspace(body);
          routeToReactUrl(paths.workspaceDetails(response.id));
          resourceQuotaBody['id'] = response.id;
        }

        if (values.resourceQuotas) {
          resourceQuotaBody['clusterQuotaPairs'] = pick(
            values.resourceQuotas,
            resourceManagers.getOrElse([]),
          );
          await setResourceQuotas(resourceQuotaBody);
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
  }, [form, workspaceId, canModifyAUG, canModifyCPS, resourceManagers]);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        form: idPrefix + FORM_ID,
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
