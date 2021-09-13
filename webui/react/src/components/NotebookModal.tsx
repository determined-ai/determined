import { Alert, Button, InputNumber, Modal } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { Dispatch, useCallback, useEffect, useReducer, useState } from 'react';

import useStorage from 'hooks/useStorage';
import { getResourcePools, getTaskTemplates } from 'services/api';
import { NotebookConfig, RawJson, ResourcePool, Template } from 'types';
import { launchNotebook, previewNotebook } from 'utils/task';

import Link from './Link';
import css from './NotebookModal.module.scss';
import Spinner from './Spinner';

const { Option } = Select;
const { Item } = Form;

const STORAGE_PATH = 'notebook-launch';
const STORAGE_KEY = 'notebook-config';
const DEFAULT_SLOT_COUNT = 1;

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
    storage.getWithDefault(STORAGE_KEY, { slots: DEFAULT_SLOT_COUNT }),
  );

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
  config?: string;
  configError?: string;
  onChange: (config: string) => void;
  setButtonDisabled: (buttonDisabled: boolean) => void;
}

interface ResourceInfo {
  hasAux: boolean;
  hasCompute: boolean;
  maxSlots: number | undefined;
}

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const NotebookModal: React.FC<NotebookModalProps> = (
  { visible = false, onLaunch, ...props }: NotebookModalProps,
) => {

  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ fields, dispatch ] = useNotebookForm();
  const [ config, setConfig ] = useState<string | undefined>();
  const [ buttonDisabled, setButtonDisabled ] = useState(false);

  const fetchConfig = useCallback(async () => {
    try {
      const newConfig = await previewNotebook(
        fields.slots,
        fields.template,
        fields.name,
        fields.pool,
      );
      setConfig(yaml.dump(newConfig));
    } catch (e) {
      setConfig(undefined);
    }
  }, [ fields ]);

  useEffect(() => {
    if (showFullConfig) fetchConfig();
  }, [ fetchConfig, showFullConfig ]);

  const handleSecondary = useCallback(() => {
    if (showFullConfig) {
      setButtonDisabled(false);
    }
    setShowFullConfig(show => !show);
  }, [ showFullConfig ]);

  const handleCreateEnvironment = useCallback(() => {
    if (showFullConfig) {
      launchNotebook(yaml.load(config || '') as RawJson);
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
  }, [ config, fields, onLaunch, showFullConfig ]);

  const handleConfigChange = useCallback((config: string) => setConfig(config), []);

  return (
    <Modal
      footer={<>
        <Button
          onClick={handleSecondary}>
          {showFullConfig ? 'Show Simple Config' : 'Show Full Config'}
        </Button>
        <Button
          disabled={buttonDisabled}
          type="primary"
          onClick={handleCreateEnvironment}>Launch</Button>
      </>}
      title="Launch JupyterLab"
      visible={visible}
      width={540}
      {...props}>
      {showFullConfig ?
        <NotebookFullConfig
          config={config}
          setButtonDisabled={setButtonDisabled}
          onChange={handleConfigChange} /> :
        <NotebookForm fields={fields} onChange={dispatch} />
      }
    </Modal>
  );
};

const NotebookFullConfig:React.FC<FullConfigProps> = (
  { config, onChange, setButtonDisabled }: FullConfigProps,
) => {
  const [ field, setField ] = useState([ { name: 'config', value: '' } ]);

  useEffect(() => {
    setField([ { name: 'config', value: config || '' } ]);
  }, [ config ]);

  const handleConfigChange = useCallback((_, allFields) => {
    if (!Array.isArray(allFields) || allFields.length === 0) return;
    try {
      const configString = allFields[0].value;
      onChange(configString);
    } catch (e) {}
  }, [ onChange ]);

  return (
    <Form
      fields={field}
      onFieldsChange={handleConfigChange}>
      <div className={css.note}>
        <Link external path="/docs/reference/api/command-notebook-config.html">
        Read about notebook settings
        </Link>
      </div>
      <React.Suspense
        fallback={<div className={css.loading}><Spinner tip="Loading text editor..." /></div>}>
        <Item
          name="config"
          rules={[
            { message: 'Notebook config required', required: true },
            {
              validator: (rule, value) => {
                try {
                  yaml.load(value);
                  setButtonDisabled(false);
                  return Promise.resolve();
                } catch (err) {
                  setButtonDisabled(true);
                  return Promise.reject(new Error(`Invalid YAML on line ${err.mark.line}.`));
                }
              },
            },
          ]}>
          <MonacoEditor
            height="40vh"
            options={{
              wordWrap: 'on',
              wrappingIndent: 'indent',
            }}
          />
        </Item>
        {!config && <Alert message="Unable to load notebook config" type="error" />}
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
    { hasAux: false, hasCompute: true, maxSlots: DEFAULT_SLOT_COUNT },
  );

  const calculateResourceInfo = useCallback((selectedPoolName: string | undefined) => {
    const selectedPool = resourcePools.find(pool => pool.name === selectedPoolName);
    if (!selectedPool) {
      return { hasAux: false, hasCompute: false, maxSlots: 0 };
    }
    const hasAuxCapacity = selectedPool.auxContainerCapacityPerAgent > 0;
    const hasComputeCapacity = selectedPool.slotsAvailable > 0
      || (!!selectedPool.slotsPerAgent && selectedPool.slotsPerAgent > 0);
    const maxSlots = hasComputeCapacity ?
      (selectedPool.slotsPerAgent && selectedPool.slotsPerAgent > 0 ?
        selectedPool.slotsPerAgent : undefined) : 0;
    if (hasAuxCapacity && !hasComputeCapacity) {
      onChange({ key: 'slots', value: 0 });
    }
    return {
      hasAux: hasAuxCapacity,
      hasCompute: hasComputeCapacity,
      maxSlots: maxSlots,
    };
  }, [ onChange, resourcePools ]);

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
      {resourceInfo.hasCompute &&
        <LabelledLine
          content = {
            <InputNumber
              defaultValue={fields.slots !== undefined ? fields.slots : DEFAULT_SLOT_COUNT}
              max={resourceInfo.maxSlots}
              min={resourceInfo.hasAux ? 0 : 1}
              value={fields.slots}
              onChange={(value) => onChange({ key: 'slots', value: value })} />}
          label="Slots" />
      }
    </div>
  );
};

export default NotebookModal;
