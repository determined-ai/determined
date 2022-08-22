import { Alert, Button, InputNumber } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalFuncProps } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Link from 'components/Link';
import useSettings, { BaseType, SettingsConfig, UpdateSettings } from 'hooks/useSettings';
import { getResourcePools, getTaskTemplates } from 'services/api';
import Spinner from 'shared/components/Spinner/Spinner';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import usePrevious from 'shared/hooks/usePrevious';
import { RawJson } from 'shared/types';
import { ResourcePool, Template } from 'types';
import handleError from 'utils/error';
import { JupyterLabOptions, launchJupyterLab, previewJupyterLab } from 'utils/jupyter';

import css from './useModalJupyterLab.module.scss';

const { Option } = Select;
const { Item } = Form;

const STORAGE_PATH = 'jupyter-lab';
const DEFAULT_SLOT_COUNT = 1;

const settingsConfig : SettingsConfig = {
  settings: [
    {
      defaultValue: '',
      key: 'name',
      skipUrlEncoding: true,
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: '',
      key: 'pool',
      skipUrlEncoding: true,
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: DEFAULT_SLOT_COUNT,
      key: 'slots',
      skipUrlEncoding: true,
      type: { baseType: BaseType.Integer },
    },
    {
      key: 'template',
      skipUrlEncoding: true,
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: STORAGE_PATH,
};

interface FormProps {
  fields: JupyterLabOptions;
  updateFields?: UpdateSettings<JupyterLabOptions>;
}

interface FullConfigProps {
  config?: string;
  configError?: string;
  onChange?: (config: string) => void;
  setButtonDisabled: (buttonDisabled: boolean) => void;
}

const MonacoEditor = React.lazy(() => import('components/MonacoEditor'));

const useModalJupyterLab = (): ModalHooks => {
  const [ visible, setVisible ] = useState(false);
  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ config, setConfig ] = useState<string>();
  const [ configError, setConfigError ] = useState<string>();
  const [ buttonDisabled, setButtonDisabled ] = useState(false);
  const previousConfig = usePrevious(config, config);
  const previousShowConfig = usePrevious(showFullConfig, showFullConfig);

  const handleModalClose = useCallback(() => setVisible(false), []);

  const { modalClose, modalOpen: openOrUpdate, ...modalHook } = useModal(
    { onClose: handleModalClose },
  );

  const { settings: fields, updateSettings: updateFields } = useSettings<JupyterLabOptions>(
    settingsConfig,
  );
  const previousFields = usePrevious(fields, undefined);

  const fetchConfig = useCallback(async () => {
    try {
      const newConfig = await previewJupyterLab({
        name: fields.name,
        pool: fields.pool,
        slots: fields.slots,
        template: fields.template,
      });
      setConfig(yaml.dump(newConfig));
    } catch (e) {
      setConfig(undefined);
      setConfigError('Unable to fetch JupyterLab config.');
    }
  }, [ fields ]);

  const handleSecondary = useCallback(() => {
    if (showFullConfig) setButtonDisabled(false);
    setShowFullConfig((show) => !show);
  }, [ showFullConfig ]);

  const handleCreateEnvironment = useCallback(() => {
    if (showFullConfig) {
      launchJupyterLab({ config: yaml.load(config || '') as RawJson });
    } else {
      launchJupyterLab({
        name: fields.name,
        pool: fields.pool,
        slots: fields.slots,
        template: fields.template,
      });
    }
    modalClose();
    setVisible(false);
  }, [ config, fields, showFullConfig, modalClose ]);

  const handleConfigChange = useCallback((config: string) => {
    setConfig(config);
    setConfigError(undefined);
  }, []);

  const formContent = useMemo(() => showFullConfig ? (
    <JupyterLabFullConfig
      config={config}
      configError={configError}
      setButtonDisabled={setButtonDisabled}
      onChange={handleConfigChange}
    />
  ) : (
    <JupyterLabForm fields={fields} updateFields={updateFields} />
  ), [ config, configError, fields, handleConfigChange, showFullConfig, updateFields ]);

  const content = useMemo(() => (
    <>
      {formContent}
      <div className={css.buttons}>
        <Button onClick={handleSecondary}>
          {showFullConfig ? 'Show Simple Config' : 'Show Full Config'}
        </Button>
        <Button
          disabled={buttonDisabled}
          type="primary"
          onClick={handleCreateEnvironment}>
          Launch
        </Button>
      </div>
    </>
  ), [ formContent, buttonDisabled, handleCreateEnvironment, handleSecondary, showFullConfig ]);

  const modalProps: ModalFuncProps = useMemo(() => ({
    className: css.noFooter,
    closable: true,
    content,
    icon: null,
    title: 'Launch JupyterLab',
    width: showFullConfig ? 1000 : undefined,
  }), [ content, showFullConfig ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setVisible(true);
    openOrUpdate({ ...modalProps, ...initialModalProps });
  }, [ modalProps, openOrUpdate ]);

  // Fetch full config when showing advanced mode.
  useEffect(() => {
    if (showFullConfig) {
      fetchConfig();
    }
  }, [ fetchConfig, showFullConfig ]);

  // Update modal when any form fields change.
  useEffect(() => {
    if (visible && fields !== previousFields) {
      openOrUpdate(modalProps);
    }
  }, [ fields, previousFields, openOrUpdate, modalProps, visible ]);

  // Update the modal when user toggles the `Show Full Config` button.
  useEffect(() => {
    if (visible && (config !== previousConfig || showFullConfig !== previousShowConfig)) {
      openOrUpdate(modalProps);
    }
  }, [
    config,
    modalProps,
    openOrUpdate,
    previousConfig,
    previousShowConfig,
    showFullConfig,
    visible,
  ]);

  return { modalClose, modalOpen, ...modalHook };
};

const JupyterLabFullConfig: React.FC<FullConfigProps> = ({
  config,
  configError,
  onChange,
  setButtonDisabled,
}: FullConfigProps) => {
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
                } catch (err: unknown) {
                  setButtonDisabled(true);
                  return Promise.reject(new Error(
                    `Invalid YAML on line ${(err as {mark: {line: string}}).mark.line}.`,
                  ));
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
      </React.Suspense>
      {configError && <Alert message={configError} type="error" />}
    </Form>
  );
};

const JupyterLabForm: React.FC<FormProps> = ({ updateFields, fields }: FormProps) => {
  const [ templates, setTemplates ] = useState<Template[]>([]);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);

  const resourceInfo = useMemo(() => {
    const selectedPool = resourcePools.find((pool) => pool.name === fields.pool);
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
    if (hasAuxCapacity && !hasComputeCapacity) updateFields?.({ slots: 0 });
    if (hasComputeCapacity && fields.slots == null) updateFields?.({ slots: DEFAULT_SLOT_COUNT });

    return {
      hasAux: hasAuxCapacity,
      hasCompute: hasComputeCapacity,
      maxSlots: maxSlots,
    };
  }, [ fields.pool, updateFields, resourcePools, fields.slots ]);

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

  useEffect(() => {
    if (!fields.pool && resourcePools[0]?.name) {
      const pool = resourcePools[0]?.name;
      updateFields?.({ pool });
    }
  }, [ resourcePools, fields.pool, updateFields ]);

  return (
    <div className={css.form}>
      {[
        {
          content: (
            <Select
              allowClear
              placeholder="No template (optional)"
              value={fields.template}
              onChange={(value) => updateFields?.({ template: value?.toString() })}>
              {templates.map((temp) => (
                <Option key={temp.name} value={temp.name}>{temp.name}</Option>
              ))}
            </Select>
          ),
          label: 'Template',
        },
        {
          content: (
            <Input
              placeholder="Name (optional)"
              value={fields.name}
              onChange={(e) => updateFields?.({ name: e.target.value })}
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
              onChange={(value) => updateFields?.({ pool: value })}>
              {resourcePools.map((pool) => (
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
              onChange={(value) => updateFields?.({ slots: value })}
            />
          ),
          label: 'Slots',
        },
      ].map((row) => {
        if (row.condition === false) return null;
        return <div className={css.line} key={row.label}><p>{row.label}</p>{row.content}</div>;
      })}
    </div>
  );
};

export default useModalJupyterLab;
