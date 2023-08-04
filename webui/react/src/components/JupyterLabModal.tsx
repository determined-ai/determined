import { Alert, Select } from 'antd';
import { number, string, undefined as undefinedType, union } from 'io-ts';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Form, { FormInstance } from 'components/kit/Form';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import { Modal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import usePermissions from 'hooks/usePermissions';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import { getTaskTemplates } from 'services/api';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { RawJson } from 'types';
import { Template, Workspace } from 'types';
import handleError from 'utils/error';
import { JupyterLabOptions, launchJupyterLab, previewJupyterLab } from 'utils/jupyter';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { useObservable } from 'utils/observable';

const { Option } = Select;

const STORAGE_PATH = 'jupyter-lab';
const DEFAULT_SLOT_COUNT = 1;

const settingsConfig: SettingsConfig<JupyterLabOptions> = {
  settings: {
    name: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'name',
      type: union([string, undefinedType]),
    },
    pool: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'pool',
      type: union([string, undefinedType]),
    },
    slots: {
      defaultValue: DEFAULT_SLOT_COUNT,
      skipUrlEncoding: true,
      storageKey: 'slots',
      type: union([number, undefinedType]),
    },
    template: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'template',
      type: union([string, undefinedType]),
    },
    workspaceId: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'workspaceId',
      type: union([number, undefinedType]),
    },
  },
  storagePath: STORAGE_PATH,
};

interface FullConfigProps {
  config: Loadable<string>;
  configError?: string;
  currentWorkspace?: Workspace;
  form: FormInstance;
  lockedWorkspace: boolean;
  onChange?: (config: string) => void;
  setWorkspace: (arg0: Workspace | undefined) => void;
  workspaces: Workspace[];
}

interface Props {
  workspace?: Workspace;
}

const CodeEditor = React.lazy(() => import('components/kit/CodeEditor'));

const JupyterLabModalComponent: React.FC<Props> = ({ workspace }: Props) => {
  const [showFullConfig, setShowFullConfig] = useState(false);
  const [config, setConfig] = useState<Loadable<string>>(NotLoaded);
  const [configError, setConfigError] = useState<string>();
  const [fullConfigFormInvalid, setFullConfigFormInvalid] = useState(true);
  const [form] = Form.useForm<JupyterLabOptions>();
  const [fullConfigForm] = Form.useForm();
  const { canCreateWorkspaceNSC } = usePermissions();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces)).filter(
    (workspace) => canCreateWorkspaceNSC({ workspace }),
  );
  const [currentWorkspace, setCurrentWorkspace] = useState<Workspace | undefined>(workspace);

  const validateFullConfigForm = useCallback(() => {
    const fields = fullConfigForm.getFieldsError();
    const hasError = fields.some((f) => f.errors.length) || !currentWorkspace;
    setFullConfigFormInvalid(hasError);
  }, [currentWorkspace, fullConfigForm]);

  const { settings: defaults, updateSettings: updateDefaults } =
    useSettings<JupyterLabOptions>(settingsConfig);

  const handleModalClose = useCallback(() => {
    const fields: JupyterLabOptions = form.getFieldsValue(true);
    updateDefaults(fields);
  }, [form, updateDefaults]);

  const fetchConfig = useCallback(async () => {
    setConfig(NotLoaded);

    const fields: JupyterLabOptions = form.getFieldsValue(true);
    try {
      const newConfig = await previewJupyterLab({
        name: fields?.name,
        pool: fields?.pool,
        slots: fields?.slots,
        template: fields?.template,
        workspaceId: fields.workspaceId,
      });
      setConfig(Loaded(yaml.dump(newConfig)));
    } catch (e) {
      setConfigError('Unable to fetch JupyterLab config.');
    }
  }, [form]);

  const handleSecondary = useCallback(() => {
    if (showFullConfig) setFullConfigFormInvalid(false);
    setShowFullConfig((show) => !show);
  }, [showFullConfig]);

  const handleSubmit = useCallback(async () => {
    const fields: JupyterLabOptions = form.getFieldsValue(true);
    updateDefaults(fields);
    if (showFullConfig) {
      const values = await fullConfigForm.validateFields();
      const usableConfig = Loadable.isLoaded(config) ? config.data : '';

      if (values) {
        launchJupyterLab({
          config: yaml.load(usableConfig) as RawJson,
          workspaceId: values.workspaceId,
        });
      }
    } else {
      const values = await form.validateFields();
      if (values) {
        launchJupyterLab({
          name: fields?.name,
          pool: fields?.pool,
          slots: fields?.slots,
          template: fields?.template,
          workspaceId: fields.workspaceId,
        });
      }
    }
  }, [config, fullConfigForm, form, showFullConfig, updateDefaults]);

  const handleConfigChange = useCallback(
    (config: string) => {
      validateFullConfigForm();
      setConfig(Loaded(config));
      setConfigError(undefined);
    },
    [validateFullConfigForm],
  );

  useEffect(validateFullConfigForm, [currentWorkspace, validateFullConfigForm]);

  useEffect(() => workspaceStore.fetch(), []);

  // Fetch full config when showing advanced mode.
  useEffect(() => {
    if (showFullConfig) {
      fetchConfig();
    }
  }, [fetchConfig, showFullConfig]);

  return (
    <Modal
      cancel
      footerLink={
        showFullConfig
          ? {
              text: 'Read about JupyterLab settings',
              url: '/docs/reference/api/command-notebook-config.html',
            }
          : undefined
      }
      size={showFullConfig ? 'large' : 'small'}
      submit={{
        disabled: showFullConfig ? fullConfigFormInvalid : !currentWorkspace?.id,
        handleError,
        handler: handleSubmit,
        text: 'Launch',
      }}
      title="Launch JupyterLab"
      onClose={handleModalClose}>
      {showFullConfig ? (
        <JupyterLabFullConfig
          config={config}
          configError={configError}
          currentWorkspace={currentWorkspace}
          form={fullConfigForm}
          lockedWorkspace={!!workspace}
          setWorkspace={setCurrentWorkspace}
          workspaces={workspaces}
          onChange={handleConfigChange}
        />
      ) : (
        <JupyterLabForm
          currentWorkspace={currentWorkspace}
          defaults={defaults}
          form={form}
          lockedWorkspace={!!workspace}
          setWorkspace={setCurrentWorkspace}
          workspaces={workspaces}
        />
      )}
      <div>
        <Button onClick={handleSecondary}>
          {showFullConfig ? 'Show Simple Config' : 'Show Full Config'}
        </Button>
      </div>
    </Modal>
  );
};

export default JupyterLabModalComponent;

const JupyterLabFullConfig: React.FC<FullConfigProps> = ({
  config,
  configError,
  currentWorkspace,
  form,
  lockedWorkspace,
  onChange,
  setWorkspace,
  workspaces,
}: FullConfigProps) => {
  const usableConfig = useMemo(() => (Loadable.isLoaded(config) ? config.data : ''), [config]);
  const [field, setField] = useState([
    { name: 'config', value: usableConfig },
    { name: 'workspaceId', value: currentWorkspace?.id },
  ]);

  const handleConfigChange = useCallback(
    (_: unknown, allFields: unknown) => {
      if (!Array.isArray(allFields) || allFields.length === 0) return;
      try {
        const configString = allFields.find((field) => field.name[0] === 'config').value;
        onChange?.(configString);
      } catch (e) {
        handleError(e);
      }
    },
    [onChange],
  );

  useEffect(() => {
    setField((curField) => [
      ...curField.filter((f) => f.name[0] === 'workspaceId'),
      { name: 'config', value: usableConfig },
    ]);
  }, [usableConfig]);

  useEffect(() => {
    form.setFieldValue('workspaceId', currentWorkspace?.id);
  }, [currentWorkspace, form]);
  useEffect(() => {
    form.setFieldValue('config', usableConfig);
  }, [usableConfig, form]);

  const onSelectWorkspace = (workspaceId?: number) => {
    const selected = workspaces.find((w) => workspaceId && w.id === workspaceId);
    setWorkspace(selected);
  };

  return (
    <Form fields={field} form={form} onFieldsChange={handleConfigChange}>
      <React.Suspense fallback={<Spinner spinning tip="Loading text editor..." />}>
        <Form.Item
          initialValue={currentWorkspace?.id}
          label="Workspace"
          name="workspaceId"
          rules={[{ message: 'Workspace is required', required: true, type: 'number' }]}>
          <Select
            allowClear
            disabled={lockedWorkspace}
            placeholder="Workspace (required)"
            onChange={onSelectWorkspace}>
            {workspaces.map((workspace: Workspace) => (
              <Option key={workspace.id} value={workspace.id}>
                {workspace.name}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          initialValue={usableConfig}
          name="config"
          rules={[
            { message: 'JupyterLab config required', required: true },
            {
              validator: (rule, value) => {
                try {
                  yaml.load(value);
                  return Promise.resolve();
                } catch (err: unknown) {
                  return Promise.reject(
                    new Error(
                      `Invalid YAML on line ${(err as { mark: { line: string } }).mark.line}.`,
                    ),
                  );
                }
              },
            },
          ]}>
          <CodeEditor
            files={[
              { get: () => Promise.resolve(Loadable.getOrElse('', config)), key: 'config.yaml' },
            ]}
            height="40vh"
            onError={handleError}
          />
        </Form.Item>
      </React.Suspense>
      {configError && <Alert message={configError} type="error" />}
    </Form>
  );
};

const JupyterLabForm: React.FC<{
  currentWorkspace?: Workspace;
  defaults: JupyterLabOptions;
  form: FormInstance<JupyterLabOptions>;
  lockedWorkspace: boolean;
  setWorkspace: (arg0: Workspace | undefined) => void;
  workspaces: Workspace[];
}> = ({ form, currentWorkspace, defaults, lockedWorkspace, setWorkspace, workspaces }) => {
  const [templates, setTemplates] = useState<Template[]>([]);

  const resourcePools = Loadable.getOrElse([], useObservable(clusterStore.resourcePools));

  const selectedPoolName = Form.useWatch('pool', form);

  const resourceInfo = useMemo(() => {
    const selectedPool = resourcePools.find((pool) => pool.name === selectedPoolName);
    if (!selectedPool) return { hasAux: false, hasCompute: false, maxSlots: 0 };

    /**
     * For static resource pools, the slots-per-agent comes through as -1,
     * meaning it is unknown how many we may have.
     */
    const hasAuxCapacity = selectedPool.auxContainerCapacityPerAgent > 0;
    const hasSlots = selectedPool.slotsAvailable > 0;
    const maxSlots = selectedPool.slotsPerAgent ?? 0;
    const hasSlotsPerAgent = maxSlots !== 0;
    const hasComputeCapacity = hasSlots || hasSlotsPerAgent;

    return {
      hasAux: hasAuxCapacity,
      hasCompute: hasComputeCapacity,
      maxSlots: maxSlots,
    };
  }, [selectedPoolName, resourcePools]);

  useEffect(() => {
    if (!resourceInfo.hasCompute && resourceInfo.hasAux) form.setFieldValue('slots', 0);
    else if (resourceInfo.hasCompute) {
      const slots = form.getFieldValue('slots');
      if (slots == null) form.setFieldValue('slots', DEFAULT_SLOT_COUNT);
    }
  }, [resourceInfo, form]);

  const fetchTemplates = useCallback(async () => {
    try {
      setTemplates(await getTaskTemplates({}));
    } catch (e) {
      handleError(e);
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  useEffect(() => {
    const fields = form.getFieldsValue(true);
    if (!fields?.pool && resourcePools[0]?.name) {
      const firstPoolInList = resourcePools[0]?.name;
      form.setFieldValue('pool', firstPoolInList);
    }
  }, [resourcePools, form]);

  useEffect(() => {
    form.setFieldValue('workspaceId', currentWorkspace?.id);
  }, [currentWorkspace, form]);

  const onSelectWorkspace = (workspaceId?: number) => {
    const selected = workspaces.find((w) => workspaceId && w.id === workspaceId);
    setWorkspace(selected);
  };

  return (
    <Form form={form}>
      <Form.Item
        initialValue={currentWorkspace?.id}
        label="Workspace"
        name="workspaceId"
        rules={[{ message: 'Workspace is required', required: true, type: 'number' }]}>
        <Select
          allowClear
          disabled={lockedWorkspace}
          placeholder="Workspace (required)"
          onChange={onSelectWorkspace}>
          {workspaces.map((workspace: Workspace) => (
            <Option key={workspace.id} value={workspace.id}>
              {workspace.name}
            </Option>
          ))}
        </Select>
      </Form.Item>
      <Form.Item initialValue={defaults?.template} label="Template" name="template">
        <Select allowClear placeholder="No template (optional)">
          {templates.map((temp) => (
            <Option key={temp.name} value={temp.name}>
              {temp.name}
            </Option>
          ))}
        </Select>
      </Form.Item>
      <Form.Item initialValue={defaults?.name} label="Name" name="name">
        <Input placeholder="Name (optional)" />
      </Form.Item>
      <Form.Item initialValue={defaults?.pool} label="Resource Pool" name="pool">
        <Select allowClear placeholder="Pick the best option" showSearch>
          {resourcePools.map((pool) => (
            <Option key={pool.name} value={pool.name}>
              {pool.name}
            </Option>
          ))}
        </Select>
      </Form.Item>
      <Form.Item
        hidden={!resourceInfo.hasCompute}
        initialValue={defaults?.slots}
        label="Slots"
        name="slots">
        <InputNumber
          max={resourceInfo.maxSlots === -1 ? Number.MAX_SAFE_INTEGER : resourceInfo.maxSlots}
          min={0}
        />
      </Form.Item>
    </Form>
  );
};
