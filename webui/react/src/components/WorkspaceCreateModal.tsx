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
import clusterStore from 'stores/cluster';
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

const WorkspaceCreateModalComponent: React.FC<Props> = ({ onClose, workspaceId }: Props = {}) => {
  const idPrefix = useId();
  const {
    canModifyWorkspaceAgentUserGroup,
    canModifyWorkspaceCheckpointStorage,
    canSetWorkspaceNamespaceBindings,
    canSetResourceQuotas,
    canViewResourceQuotas,
    canModifyWorkspace,
  } = usePermissions();
  const info = useObservable(determinedStore.info);
  const [form] = Form.useForm<FormInputs>();
  const useAgentUser = Form.useWatch('useAgentUser', form);
  const useAgentGroup = Form.useWatch('useAgentGroup', form);
  const useCheckpointStorage = Form.useWatch('useCheckpointStorage', form);
  const watchBindings = Form.useWatch('bindings', form);
  const loadableResourceManagers = useObservable(clusterStore.kubernetesResourceManagers);
  const resourceManagers = Loadable.getOrElse([], loadableResourceManagers);
  const namespaceBindingsList = useAsync(
    async (canceller) => {
      if (
        workspaceId === undefined ||
        resourceManagers.length === 0 ||
        !canSetWorkspaceNamespaceBindings
      ) {
        return NotLoaded;
      }
      try {
        const clusterNamespacePairs = await listWorkspaceNamespaceBindings(
          { id: workspaceId },
          { signal: canceller.signal },
        );
        return clusterNamespacePairs.namespaceBindings;
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Failed to fetch list of workspace namespace bindings.',
          silent: false,
          type: ErrorType.Server,
        });
        return NotLoaded;
      }
    },
    [canSetWorkspaceNamespaceBindings, resourceManagers, workspaceId],
  );
  const resourceQuotasList = useAsync(
    async (canceller) => {
      if (workspaceId === undefined || resourceManagers.length === 0 || !canViewResourceQuotas) {
        return NotLoaded;
      }
      try {
        const resp = await getKubernetesResourceQuotas(
          { id: workspaceId },
          { signal: canceller.signal },
        );
        return resp.resourceQuotas;
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Failed to fetch kubernetes resource quotas for the workspace.',
          silent: false,
          type: ErrorType.Server,
        });
        return NotLoaded;
      }
    },
    [canViewResourceQuotas, resourceManagers, workspaceId],
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
  const isViewing = !!workspace && !canModifyWorkspace({ workspace });
  const isEditing = workspaceId !== undefined;

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
    if (isEditing && loadableWorkspace === NotLoaded) return <Spinner spinning />;
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
          <Input disabled={isViewing} maxLength={80} />
        </Form.Item>
        {canSetWorkspaceNamespaceBindings && resourceManagers.length > 0 && (
          <>
            <Divider />
            Namespace Bindings
            <Body inactive>
              Note: If you leave the Namespace name blank, the workspace will be bound to the
              default Namespace configured in the Master Config.
            </Body>
            <>
              {resourceManagers.map((name) => (
                <Fragment key={name}>
                  <Form.Item label={`Cluster Name: ${name}`} name={['bindings', name, 'namespace']}>
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
                      {canSetResourceQuotas && (
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
                      )}
                    </>
                  )}
                </Fragment>
              ))}
            </>
          </>
        )}
        {!canSetWorkspaceNamespaceBindings &&
          canViewResourceQuotas &&
          isEditing &&
          info.branding === BrandingType.HPE &&
          resourceManagers.length > 0 && (
            <>
              <Divider />
              Resource Quotas
              <>
                {resourceManagers.map((name) => (
                  <Form.Item
                    key={name}
                    label={`Cluster Name: ${name}`}
                    name={['resourceQuotas', name]}>
                    <Input disabled />
                  </Form.Item>
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
    loadableWorkspace,
    form,
    idPrefix,
    isEditing,
    isViewing,
    canSetWorkspaceNamespaceBindings,
    resourceManagers,
    canViewResourceQuotas,
    info.branding,
    canModifyAUG,
    useAgentUser,
    useAgentGroup,
    canModifyCPS,
    useCheckpointStorage,
    watchBindings,
    canSetResourceQuotas,
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

        if (isEditing) {
          await patchWorkspace({ id: workspaceId, ...body });
          workspaceStore.fetch(undefined, true);
          resourceQuotaBody['id'] = workspaceId;
        } else {
          const response = await workspaceStore.createWorkspace(body);
          routeToReactUrl(paths.workspaceDetails(response.id));
          resourceQuotaBody['id'] = response.id;
        }

        if (
          canSetResourceQuotas &&
          values.resourceQuotas &&
          Object.keys(values.resourceQuotas).length !== 0
        ) {
          resourceQuotaBody['clusterQuotaPairs'] = pick(values.resourceQuotas, resourceManagers);
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
  }, [
    form,
    canModifyAUG,
    canModifyCPS,
    canSetResourceQuotas,
    isEditing,
    workspaceId,
    resourceManagers,
  ]);

  return (
    <Modal
      cancel
      size="medium"
      submit={
        isViewing
          ? undefined
          : {
              form: idPrefix + FORM_ID,
              handleError,
              handler: handleSubmit,
              text: 'Save Workspace',
            }
      }
      title={isViewing ? 'Workspace Config' : `${isEditing ? 'Edit' : 'New'} Workspace`}
      onClose={() => {
        initFields(undefined);
        onClose?.();
      }}>
      {modalContent}
    </Modal>
  );
};

export default WorkspaceCreateModalComponent;
