import { Alert, Button, InputNumber } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalFuncProps } from 'antd';
import yaml from 'js-yaml';
import React, { Dispatch, useCallback, useEffect, useMemo, useReducer, useState } from 'react';

import Link from 'components/Link';
import Spinner from 'components/Spinner';
import useJupyterLab from 'hooks/useJupyterLab';
import usePrevious from 'hooks/usePrevious';
import useStorage from 'hooks/useStorage';
import { getResourcePools, getTaskTemplates } from 'services/api';
import { JupyterLabConfig, RawJson, ResourcePool, Template } from 'types';
import handleError from 'utils/error';

import css from './useJupyterLabModal.module.scss';
import useModal, { ModalHooks } from './useModal';

const { Option } = Select;
const { Item } = Form;

const STORAGE_PATH = 'jupyter-lab-launch';
const STORAGE_KEY = 'jupyter-lab-config';
const DEFAULT_SLOT_COUNT = 1;

type DispatchFunction = Dispatch<{
  key: keyof JupyterLabConfig,
  value: string | number | undefined
}>;

function reducer(
  state: JupyterLabConfig,
  action: { key: keyof JupyterLabConfig, value: string | number | undefined },
): JupyterLabConfig {
  return { ...state, [action.key]: action.value };
}

const useJupyterLabForm = (): [ JupyterLabConfig, DispatchFunction ] => {
  const storage = useStorage(STORAGE_PATH);
  const [ state, dispatch ] = useReducer(
    reducer,
    storage.getWithDefault(STORAGE_KEY, { slots: DEFAULT_SLOT_COUNT }),
  );

  const storeConfig = useCallback((values: JupyterLabConfig) => {
    const { name, ...storedValues } = values;
    storage.set(STORAGE_KEY, storedValues);
  }, [ storage ]);

  useEffect(() => {
    storeConfig(state);
  }, [ state, storeConfig ]);

  return [ state, dispatch ];
};
interface FormProps {
  fields: JupyterLabConfig;
  onChange?: DispatchFunction;
}

interface FullConfigProps {
  config?: string;
  configError?: string;
  onChange?: (config: string) => void;
  setButtonDisabled: (buttonDisabled: boolean) => void;
}

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const useJupyterLabModal = (): ModalHooks => {
  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal();

  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ config, setConfig ] = useState<string | undefined>();
  const previousConfig = usePrevious(config, config);
  const previousShowConfig = usePrevious(showFullConfig, showFullConfig);
  const [ buttonDisabled, setButtonDisabled ] = useState(false);

  const [ fields, dispatch ] = useJupyterLabForm();
  const { launchJupyterLab, previewJupyterLab } = useJupyterLab();

  const fetchConfig = useCallback(async () => {
    try {
      const newConfig = await previewJupyterLab({
        name: fields.name,
        pool: fields.pool,
        slots: fields.slots,
        templateName: fields.template,
      });
      setConfig(yaml.dump(newConfig));
    } catch (e) {
      setConfig(undefined);
    }
  }, [ fields, previewJupyterLab ]);

  const handleSecondary = useCallback(() => {
    if (showFullConfig) {
      setButtonDisabled(false);
    }
    setShowFullConfig(show => !show);
  }, [ showFullConfig ]);

  const handleCreateEnvironment = useCallback(() => {
    if (showFullConfig) {
      launchJupyterLab({ config: yaml.load(config || '') as RawJson });
    } else {
      launchJupyterLab({
        name: fields.name,
        pool: fields.pool,
        slots: fields.slots,
        templateName: fields.template,
      });
    }
    modalClose();
  }, [ config, fields, launchJupyterLab, showFullConfig, modalClose ]);

  const handleConfigChange = useCallback((config: string) => setConfig(config), []);
  const formContent = useMemo(() => showFullConfig ? (
    <JupyterLabFullConfig
      config={config}
      setButtonDisabled={setButtonDisabled}
      onChange={handleConfigChange}
    />
  ) : (
    <JupyterLabForm fields={fields} onChange={dispatch} />
  ), [ config, dispatch, fields, handleConfigChange, showFullConfig ]);

  const content = useMemo(() => (
    <div className={css.modalContent}>
      {formContent}
      <div className={css.buttons}>
        <Button
          onClick={handleSecondary}>
          {showFullConfig ? 'Show Simple Config' : 'Show Full Config'}
        </Button>
        <Button
          disabled={buttonDisabled}
          type="primary"
          onClick={handleCreateEnvironment}>Launch
        </Button>
      </div>
    </div>
  ), [ formContent, buttonDisabled, handleCreateEnvironment, handleSecondary, showFullConfig ]);

  const modalProps: ModalFuncProps = useMemo(
    () => (
      {
        className: css.noFooter,
        content: content,
        title: 'Launch JupyterLab',
        width: 540,
      })
    , [ content ],
  );

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  /**
   * Update the modal when user toggles the `Show Full Config` button.
   */
  useEffect(() => {
    if(config !== previousConfig || showFullConfig !== previousShowConfig){
      openOrUpdate(modalProps);
    }
  }, [ config, modalProps, openOrUpdate, previousConfig, previousShowConfig, showFullConfig ]);

  useEffect(() => {
    if (showFullConfig) fetchConfig();
  }, [ fetchConfig, showFullConfig ]);

  return { modalClose, modalOpen, modalRef };
};

const JupyterLabFullConfig: React.FC<FullConfigProps> = (
  { config, onChange, setButtonDisabled }: FullConfigProps,
) => {
  const [ field, setField ] = useState([ { name: 'config', value: '' } ]);

  const handleConfigChange = useCallback((_, allFields) => {
    if (!Array.isArray(allFields) || allFields.length === 0) return;
    try {
      const configString = allFields[0].value;
      onChange?.(configString);
    } catch (e) { handleError(e); }
  }, [ onChange ]);

  useEffect(() => {
    setField([ { name: 'config', value: config || '' } ]);
  }, [ config ]);

  return (
    <Form
      fields={field}
      onFieldsChange={handleConfigChange}>
      <div className={css.note}>
        <Link external path="/docs/reference/api/command-notebook-config.html">
          Read about JupyterLab settings
        </Link>
      </div>
      <React.Suspense
        fallback={<div className={css.loading}><Spinner tip="Loading text editor..." /></div>}>
        <Item
          name="config"
          rules={[
            { message: 'JupyterLab config required', required: true },
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
        {!config && <Alert message="Unable to load JupyterLab config" type="error" />}
      </React.Suspense>
    </Form>
  );
};

const JupyterLabForm: React.FC<FormProps> = (
  { onChange, fields }: FormProps,
) => {
  const [ templates, setTemplates ] = useState<Template[]>([]);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);

  const resourceInfo = useMemo(() => {
    const selectedPool = resourcePools.find(pool => pool.name === fields.pool);
    if (!selectedPool) return { hasAux: false, hasCompute: false, maxSlots: 0 };

    /*
     * For static resource pools, the slots-per-agent comes through as -1,
     * meaning it is unknown how many we may have.
     */
    const hasAuxCapacity = selectedPool.auxContainerCapacityPerAgent > 0;
    const hasSlots = selectedPool.slotsAvailable > 0;
    const maxSlots = selectedPool.slotsPerAgent ?? 0;
    const hasSlotsPerAgent = maxSlots !== 0;
    const hasComputeCapacity = hasSlots || hasSlotsPerAgent;
    if (hasAuxCapacity && !hasComputeCapacity) onChange?.({ key: 'slots', value: 0 });

    return {
      hasAux: hasAuxCapacity,
      hasCompute: hasComputeCapacity,
      maxSlots: maxSlots,
    };
  }, [ fields.pool, onChange, resourcePools ]);

  const fetchResourcePools = useCallback(async () => {
    try {
      setResourcePools(await getResourcePools({}));
    } catch (e) { handleError(e); }
  }, []);

  const fetchTemplates = useCallback(async () => {
    try {
      setTemplates(await getTaskTemplates({}));
    } catch (e) { handleError(e); }
  }, []);

  useEffect(() => {
    fetchResourcePools();
  }, [ fetchResourcePools ]);

  useEffect(() => {
    fetchTemplates();
  }, [ fetchTemplates ]);

  return (
    <div className={css.form}>
      {[
        {
          content: (
            <Select
              allowClear
              placeholder="No template (optional)"
              value={fields.template}
              onChange={value => onChange?.({ key: 'template', value: value?.toString() })}>
              {templates.map(temp => (
                <Option key={temp.name} value={temp.name}>{temp.name}</Option>
              ))}
            </Select>
          ),
          label: 'Template',
        },
        {
          content: (
            <Input
              placeholder="Name"
              value={fields.name}
              onChange={(e) => onChange?.({ key: 'name', value: e.target.value })}
            />
          ),
          label: 'Name',
        },
        {
          content: (
            <Select
              allowClear
              placeholder="Pick the best option"
              value={fields.pool}
              onChange={value => onChange?.({ key: 'pool', value: value })}>
              {resourcePools.map(pool => (
                <Option key={pool.name} value={pool.name}>{pool.name}</Option>
              ))}
            </Select>
          ),
          label: 'Resource Pool',
        },
        {
          condition: resourceInfo.hasCompute,
          content: (
            <InputNumber
              defaultValue={fields.slots !== undefined ? fields.slots : DEFAULT_SLOT_COUNT}
              max={resourceInfo.maxSlots === -1 ? Number.MAX_SAFE_INTEGER : resourceInfo.maxSlots}
              min={resourceInfo.hasAux ? 0 : 1}
              value={fields.slots}
              onChange={(value) => onChange?.({ key: 'slots', value: value })}
            />
          ),
          label: 'Slots',
        },
      ].map(row => {
        if (row.condition === false) return null;
        return <div className={css.line} key={row.label}><p>{row.label}</p>{row.content}</div>;
      })}
    </div>
  );
};

export default useJupyterLabModal;
