import { Button, InputNumber, Modal } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';

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

interface Props extends ModalProps {
  onLaunch?: () => void;
  visible?: boolean;
}

const resourcePools = [ {
  agentDockerImage: '',
  agentDockerNetwork: '',
  agentDockerRuntime: '',
  agentFluentImage: '',
  containerStartupScript: '',
  cpuContainerCapacity: 100,
  cpuContainerCapacityPerAgent: 100,
  cpuContainersRunning: 0,
  defaultCpuPool: true,
  defaultGpuPool: true,
  description: '',
  details: {
    aws: null,
    gcp: null,
    priorityScheduler: null,
  },
  imageId: '',
  instanceType: '',
  location: 'on-prem',
  masterCertName: '',
  masterUrl: '',
  maxAgents: 0,
  maxAgentStartingPeriod: 0,
  maxIdleAgentPeriod: 0,
  minAgents: 0,
  name: 'both',
  numAgents: 1,
  preemptible: false,
  schedulerFittingPolicy: 'FITTING_POLICY_BEST',
  schedulerType: 'SCHEDULER_TYPE_FAIR_SHARE',
  slotsAvailable: 1,
  slotsPerAgent: -1,
  slotsUsed: 0,
  startupScript: '',
  type: 'RESOURCE_POOL_TYPE_STATIC',
}, {
  agentDockerImage: '',
  agentDockerNetwork: '',
  agentDockerRuntime: '',
  agentFluentImage: '',
  containerStartupScript: '',
  cpuContainerCapacity: 100,
  cpuContainerCapacityPerAgent: 100,
  cpuContainersRunning: 0,
  defaultCpuPool: true,
  defaultGpuPool: true,
  description: '',
  details: {
    aws: null,
    gcp: null,
    priorityScheduler: null,
  },
  imageId: '',
  instanceType: '',
  location: 'on-prem',
  masterCertName: '',
  masterUrl: '',
  maxAgents: 0,
  maxAgentStartingPeriod: 0,
  maxIdleAgentPeriod: 0,
  minAgents: 0,
  name: 'cpu',
  numAgents: 1,
  preemptible: false,
  schedulerFittingPolicy: 'FITTING_POLICY_BEST',
  schedulerType: 'SCHEDULER_TYPE_FAIR_SHARE',
  slotsAvailable: -1,
  slotsPerAgent: -1,
  slotsUsed: 0,
  startupScript: '',
  type: 'RESOURCE_POOL_TYPE_STATIC',
} ];

const NotebookModal: React.FC<Props> = (
  { visible = false, onLaunch, ...props }: Props,
) => {
  const storage = useStorage(STORAGE_PATH);
  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ templates, setTemplates ] = useState<Template[]>([]);
  //const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ dummy, setDummy ] = useState(false);

  const [ resourceType, setResourceType ] = useState<ResourceType | undefined>(
    storage.getWithDefault(STORAGE_KEY, { type: undefined }).type,
  );
  const [ form ] = Form.useForm();

  const calculateResourceInfo = (selectedPoolName: string) => {
    const selectedPool = resourcePools.find(pool => pool.name === selectedPoolName);
    if (!selectedPool) {
      return { hasCPU: false, hasGPU: false, showResourceType: true };
    }
    const hasCPUCapacity = selectedPool.cpuContainerCapacityPerAgent > 0;
    const hasGPUCapacity = selectedPool.slotsAvailable > 0 || selectedPool.slotsPerAgent > 0;
    return {
      hasCPU: hasCPUCapacity,
      hasGPU: hasGPUCapacity,
      showResourceType: hasCPUCapacity && hasGPUCapacity,
    };
  };

  const { showResourceType, hasCPU, hasGPU } = calculateResourceInfo(form.getFieldValue('pool'));

  const fetchTemplates = useCallback(async () => {
    try {
      setTemplates(await getTaskTemplates({}));
    } catch {}
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [ fetchTemplates ]);
  /*
  const fetchResourcePools = useCallback(async () => {
    try {
      setResourcePools(await getResourcePools({}));
    } catch {}
  }, []);

  useEffect(() => {
    fetchResourcePools();
  }, [ fetchResourcePools ]);
  */

  const fetchConfig = useCallback(async () => {
    try {
      const values: NotebookConfig = form.getFieldsValue(true);
      const config = await previewNotebook(
        (hasCPU && !hasGPU) ? 0 : values.slots,
        values.template,
        values.name,
        values.pool,
      );
      form.setFieldsValue({ config: yaml.dump(config) });
    } catch {}
  }, [ form, hasCPU, hasGPU ]);

  useEffect(() => {
    if (showFullConfig) fetchConfig();
  }, [ showFullConfig, fetchConfig ]);

  const storeConfig = useCallback((_, values: NotebookConfig) => {
    delete values.name;
    storage.set(STORAGE_KEY, values);
  }, [ storage ]);

  const handleSecondary = useCallback(async () => {
    if (showFullConfig) {
      setShowFullConfig(false);
    } else {
      try {
        await form.validateFields();
        setShowFullConfig(true);
      } catch (e) {}
    }
  }, [ form, showFullConfig ]);

  const handleCreateEnvironment = useCallback(
    (values) => {
      if (showFullConfig) {
        launchNotebook(yaml.load(form.getFieldValue('config')) as RawJson);
      } else {

        launchNotebook(
          undefined,
          (hasCPU && !hasGPU) ? 0 : values.slots,
          values.template,
          values.name,
          values.pool,
        );
      }
      if (onLaunch) onLaunch();
    },
    [ showFullConfig, form, onLaunch, hasCPU, hasGPU ],
  );

  const handleTypeUpdate = useCallback((selectedResourceType) => {
    setResourceType(selectedResourceType as ResourceType);
  }, []);

  return <Modal
    footer={<>
      <Button onClick={handleSecondary}>{showFullConfig ? 'Edit Form' : 'Edit Full Config'}</Button>
      <Button
        type="primary"
        onClick={async () => {
          try {
            const values = await form.validateFields();
            handleCreateEnvironment(values);
          } catch (e) {}
        }
        }>Launch</Button>
    </>}
    title="Launch JupyterLab"
    visible={visible}
    width={540}
    {...props}>
    {showFullConfig ?
      <Form form={form}>
        <div className={css.note}>
          <Link external path="/docs/reference/command-notebook-config.html">
          Read about notebook settings
          </Link>
        </div>
        <React.Suspense fallback={<div className={css.loading}><Spinner /></div>}>
          <Item
            name="config"
            noStyle
            rules={[ { message: 'Invalid YAML', required: true }, () => ({
              validator(_, value) {
                try {
                  yaml.load(value);
                  return Promise.resolve();
                } catch(err) {
                  return Promise.reject(new Error('Invalid YAML'));
                }
              },
            }) ]}>
            <MonacoEditor
              height={430}
              language="yaml"
              options={{
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                selectOnLineNumbers: true,
              }} />
          </Item>
        </React.Suspense>
      </Form> :
      <Form
        form={form}
        initialValues={storage.getWithDefault(STORAGE_KEY, {
          pool: undefined,
          slots: 1,
          template: undefined,
          type: undefined,
        })}
        labelCol={{ span: 8 }}
        onValuesChange={storeConfig}>
        <Item label="Notebook Template" name="template">
          <Select allowClear placeholder="No template (optional)">
            {templates.map(temp =>
              <Option key={temp.name} value={temp.name}>{temp.name}</Option>)}
          </Select>
        </Item>
        <Item label="Name" name="name">
          <Input placeholder="Name" />
        </Item>
        <Item
          label="Resource Pool"
          name="pool">
          <Select
            allowClear
            placeholder="Pick the best option"
            onChange={() => setDummy(d => !d)}>
            {resourcePools.map(pool =>
              <Option key={pool.name} value={pool.name}>{pool.name}</Option>)}
          </Select>
        </Item>
        {showResourceType && <Item
          label="Type"
          name="type">
          <RadioGroup
            options={[ { id: ResourceType.CPU, label: ResourceType.CPU },
              { id: ResourceType.GPU, label: ResourceType.GPU } ]}
            onChange={handleTypeUpdate} />
        </Item>}
        {resourceType === 'GPU' ?
          <Item
            initialValue={1}
            label="Number of Slots"
            name="slots">
            <InputNumber min={1} />
          </Item> : null
        }
      </Form>
    }
  </Modal>;
};

export default NotebookModal;
