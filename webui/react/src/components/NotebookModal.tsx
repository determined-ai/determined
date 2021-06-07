import { Alert, Button, InputNumber, Modal } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { Dispatch, useCallback, useEffect, useReducer, useState } from 'react';

import useStorage from 'hooks/useStorage';
import { getResourcePools, getTaskTemplates } from 'services/api';
import { NotebookConfig, RawJson, ResourcePool, ResourceType, Template } from 'types';
import { launchNotebook, previewNotebook } from 'utils/task';

import Link from './Link';
import css from './NotebookModal.module.scss';
import RadioGroup from './RadioGroup';
import Spinner from './Spinner';

const MonacoEditor = React.lazy(() => import('react-monaco-editor'));

const { Option } = Select;
const { Item } = Form;

const STORAGE_PATH = 'notebook-launch';
const STORAGE_KEY = 'notebook-config';

type DispatchFunction =
  (Dispatch<{
    key: keyof NotebookConfig,
    value: string | number | undefined
  }>)

function reducer(
  state: NotebookConfig,
  action: {key: keyof NotebookConfig, value: string | number | undefined},
): NotebookConfig {
  return { ...state, [action.key]: action.value };
}

const useNotebookForm = (): [NotebookConfig, DispatchFunction] => {
  const storage = useStorage(STORAGE_PATH);
  const [ state, dispatch ] = useReducer(
    reducer,
    storage.getWithDefault(STORAGE_KEY, { slots: 1 }),
  );

  useEffect(() => {
    if (state.type === ResourceType.GPU && (state.slots === undefined || state.slots < 1)) {
      state.slots = 1;
    }
  }, [ state ]);

  const storeConfig = useCallback((values: NotebookConfig) => {
    const { name, ...storedValues } = values;
    storage.set(STORAGE_KEY, storedValues);
  }, [ storage ]);

  useEffect(() => {
    storeConfig(state);
  }, [ state, storeConfig ]);

  return [ state, dispatch ];
};

interface NotebookModalProps extends ModalProps {
  onLaunch?: () => void;
  visible?: boolean;
}

interface FormProps {
  fields: NotebookConfig;
  onChange: DispatchFunction;
}

interface FullConfigProps {
  config?: RawJson;
  configError?: string;
  onChange: (config: RawJson) => void;
}

interface ResourceInfo {
  hasCPU: boolean;
  hasGPU: boolean;
  showResourceType: boolean;
}

const NotebookModal: React.FC<NotebookModalProps> = (
  { visible = false, onLaunch, ...props }: NotebookModalProps,
) => {

  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ fields, dispatch ] = useNotebookForm();
  const [ config, setConfig ] = useState<RawJson | undefined>();

  const fetchConfig = useCallback(async () => {
    try {
      const newConfig = await previewNotebook(
        fields.slots,
        fields.template,
        fields.name,
        fields.pool,
      );
      setConfig(newConfig);
    } catch (e) {
      setConfig(undefined);
    }
  }, [ fields ]);

  useEffect(() => {
    if (showFullConfig) fetchConfig();
  }, [ showFullConfig, fetchConfig ]);

  const handleSecondary = useCallback(() => {
    setShowFullConfig(show => !show);
  }, []);

  const handleCreateEnvironment = useCallback(() => {
    if (showFullConfig) {
      launchNotebook(config);
    } else {
      launchNotebook(
        undefined,
        fields.slots,
        fields.template,
        fields.name,
        fields.pool,
      );
    }
    if (onLaunch) onLaunch();
  }, [ showFullConfig, onLaunch, fields, config ]);

  const handleConfigChange = useCallback((config: RawJson) => setConfig(config), []);

  return (
    <Modal
      footer={<>
        <Button
          onClick={handleSecondary}>{showFullConfig ? 'Edit Form' : 'Edit Full Config'}</Button>
        <Button
          type="primary"
          onClick={handleCreateEnvironment}>Launch</Button>
      </>}
      title="Launch JupyterLab"
      visible={visible}
      width={540}
      {...props}>
      {showFullConfig ?
        <NotebookFullConfig config={config} onChange={handleConfigChange} /> :
        <NotebookForm fields={fields} onChange={dispatch} />
      }
    </Modal>
  );
};

const NotebookFullConfig:React.FC<FullConfigProps> = (
  { config, onChange }: FullConfigProps,
) => {
  const [ field, setField ] = useState([ { name: 'config', value: '' } ]);

  useEffect(() => {
    setField([ { name: 'config', value: yaml.dump(config) } ]);
  }, [ config ]);

  const handleConfigChange = useCallback((_, allFields) => {
    if (!Array.isArray(allFields) || allFields.length === 0) return;
    try {
      const configString = allFields[0].value;
      const config = yaml.load(configString) as RawJson;
      onChange(config);
    } catch (e) {}
  }, [ onChange ]);

  return (
    <Form
      fields={field}
      onFieldsChange={handleConfigChange}>
      <div className={css.note}>
        <Link external path="/docs/reference/command-notebook-config.html">
        Read about notebook settings
        </Link>
      </div>
      <React.Suspense fallback={<div className={css.loading}><Spinner /></div>}>
        <Item
          name="config"
          rules={[
            { message: 'Notebook config required!', required: true },
            {
              validator: (rule, value) => {
                try {
                  yaml.load(value);
                  return Promise.resolve();
                } catch (err) {
                  return Promise.reject(new Error(`Invalid YAML on line ${err.mark.line}.`));
                }
              },
            },
          ]}>
          <MonacoEditor
            height={430}
            language="yaml"
            options={{
              minimap: { enabled: false },
              scrollBeyondLastLine: false,
              selectOnLineNumbers: true,
            }}
          />
        </Item>
        {!config && <Alert message="Unable to load notebook config." type="error" />}
      </React.Suspense>
    </Form>
  );
};

interface LabelledLineProps {
  content: JSX.Element;
  label: string;
}

const LabelledLine: React.FC<LabelledLineProps> = (
  { label, content }: LabelledLineProps,
) => {
  return <div className={css.line}><p>{label}</p>{content}</div>;
};

const NotebookForm:React.FC<FormProps> = (
  { onChange, fields }: FormProps,
) => {
  const [ templates, setTemplates ] = useState<Template[]>([]);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ resourceInfo, setResourceInfo ] = useState<ResourceInfo>(
    { hasCPU: false, hasGPU: true, showResourceType: true },
  );

  const calculateResourceInfo = useCallback((selectedPoolName: string | undefined) => {
    const selectedPool = resourcePools.find(pool => pool.name === selectedPoolName);
    if (!selectedPool) {
      return { hasCPU: false, hasGPU: false, showResourceType: true };
    }
    const hasCPUCapacity = selectedPool.cpuContainerCapacityPerAgent > 0;
    const hasGPUCapacity = selectedPool.slotsAvailable > 0
      || (!!selectedPool.slotsPerAgent && selectedPool.slotsPerAgent > 0);
    if (hasCPUCapacity && !hasGPUCapacity) {
      onChange({ key: 'type', value: ResourceType.CPU });
      onChange({ key: 'slots', value: 0 });
    } else if (!hasCPUCapacity && hasGPUCapacity) {
      onChange({ key: 'type', value: ResourceType.GPU });
    }
    return {
      hasCPU: hasCPUCapacity,
      hasGPU: hasGPUCapacity,
      showResourceType: hasCPUCapacity && hasGPUCapacity,
    };
  }, [ resourcePools, onChange ]);

  useEffect(() => {
    setResourceInfo(calculateResourceInfo(fields.pool));
  }, [ fields.pool, calculateResourceInfo ]);

  const fetchTemplates = useCallback(async () => {
    try {
      setTemplates(await getTaskTemplates({}));
    } catch {}
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [ fetchTemplates ]);

  const fetchResourcePools = useCallback(async () => {
    try {
      setResourcePools(await getResourcePools({}));
    } catch {}
  }, []);

  useEffect(() => {
    fetchResourcePools();
  }, [ fetchResourcePools ]);

  return (
    <div className={css.form}>
      <LabelledLine
        content = {
          <Select
            allowClear
            placeholder="No template (optional)"
            value={fields.template}
            onChange={(value) => onChange({ key: 'template', value: value?.toString() })}>
            {templates.map(temp =>
              <Option key={temp.name} value={temp.name}>{temp.name}</Option>)}
          </Select>}
        label="Template" />
      <LabelledLine
        content = {
          <Input
            placeholder="Name"
            value={fields.name}
            onChange={(e) => onChange({ key: 'name', value: e.target.value })} />}
        label="Name" />
      <LabelledLine
        content = {
          <Select
            allowClear
            placeholder="Pick the best option"
            value={fields.pool}
            onChange={(value) => onChange({ key: 'pool', value: value })}>
            {resourcePools.map(pool =>
              <Option key={pool.name} value={pool.name}>{pool.name}</Option>)}
          </Select>}
        label="Resource Pool" />
      {resourceInfo.showResourceType &&
    <LabelledLine
      content = {
        <RadioGroup
          options={[ { id: ResourceType.CPU, label: ResourceType.CPU },
            { id: ResourceType.GPU, label: ResourceType.GPU } ]}
          value={fields.type}
          onChange={(value) => {
            onChange({ key: 'type', value: value });
            onChange({ key: 'slots', value: value === ResourceType.CPU? 0: fields.slots });
          }} />}
      label="Type" />}
      {fields.type === ResourceType.GPU &&
    <LabelledLine
      content = {
        <InputNumber
          defaultValue={1}
          min={1}
          value={fields.slots}
          onChange={(value) => onChange({ key: 'slots', value: value })} />}
      label="Slots" />
      }
    </div>
  );
};

export default NotebookModal;
