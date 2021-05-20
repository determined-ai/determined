import { Button, Col, InputNumber, Modal, Row } from 'antd';
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

const NotebookModal: React.FC<Props> = (
  { visible = false, onLaunch, ...props }: Props,
) => {
  const storage = useStorage(STORAGE_PATH);
  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ templates, setTemplates ] = useState<Template[]>([]);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ resourceTypeOptions, setResourceTypeOptions ] =
    useState<{id:ResourceType, label:ResourceType}[]>(
      [ { id:ResourceType.CPU, label:ResourceType.CPU },
        { id:ResourceType.GPU, label:ResourceType.GPU } ],
    );
  const [ resourceType, setResourceType ] =
    useState(storage.getWithDefault(STORAGE_KEY, { type: undefined }).type);
  const [ form ] = Form.useForm();

  const fetchTemplates = useCallback(async () => {
    try {
      setTemplates(await getTaskTemplates({}));
    } catch {}
  },[]);

  useEffect(() => {
    fetchTemplates();
  }, [ fetchTemplates ]);

  const fetchResourcePools = useCallback(async () => {
    try {
      setResourcePools( await getResourcePools({}));
    } catch {}
  }, []);

  useEffect(() => {
    fetchResourcePools();
  }, [ fetchResourcePools ]);

  const fetchConfig = useCallback(async () => {
    try {
      const values: NotebookConfig = form.getFieldsValue(true);
      const config = await previewNotebook(
        values.type === 'CPU'? 0 : values.slots,
        values.template === 'default'? undefined : values.template,
        values.name,
        values.pool,
      );
      form.setFieldsValue({ config: yaml.dump(config) });
    } catch {}
  }, [ form ]);

  useEffect(()=> {
    if (showFullConfig){
      fetchConfig();
    }
  }, [ showFullConfig, fetchConfig ]);

  const storeConfig = useCallback((_, values: NotebookConfig) => {
    delete values.name;
    storage.set(STORAGE_KEY,values);
  }, [ storage ]);

  const handleSecondary = useCallback(() => {
    if (showFullConfig) {
      setShowFullConfig(show => !show);
    } else {
      form.validateFields().then(() => setShowFullConfig(show => !show)).catch();
    }
  },[ form, showFullConfig ]);

  const handleCreateEnvironment = useCallback(
    (values) =>{
      if (showFullConfig) {
        launchNotebook(yaml.load(form.getFieldValue('config')) as RawJson);
      } else {
        launchNotebook(
          undefined,
          values.resourceType === 'CPU'? 0 : values.slots,
          values.template === 'default'? undefined : values.template,
          values.name,
          values.pool,
        );
      }
      if (onLaunch) onLaunch();
    },
    [ showFullConfig, form, onLaunch ],
  );

  const handleResourcePoolUpdate = useCallback((e) => {
    if (e === '') {
      setResourceTypeOptions([ { id:ResourceType.CPU, label:ResourceType.CPU },
        { id:ResourceType.GPU, label:ResourceType.GPU } ]);
    } else {
      const pool = resourcePools.find(pool => pool.name === e);
      if (pool){
        const options = [];
        if (pool.cpuContainerCapacityPerAgent > 0) {
          options.push({ id:ResourceType.CPU, label:ResourceType.CPU });
        }
        if (pool.slotsPerAgent && pool.slotsPerAgent > 0) {
          options.push({ id:ResourceType.GPU, label:ResourceType.GPU });
        }
        setResourceTypeOptions(options);
        form.setFieldsValue({ type: undefined });
        setResourceType(undefined);
      }
    }
  },[ resourcePools, form ]);

  const handleTypeUpdate = useCallback((e) => {
    setResourceType(e);
  },[]);

  return <Modal
    footer={<>
      <Button onClick={handleSecondary}>{showFullConfig ? 'Edit Form' : 'Edit Full Config'}</Button>
      <Button
        type="primary"
        onClick={() => {
          form.validateFields().then(values => {
            handleCreateEnvironment(values);
          }).catch();
        }
        }>Launch</Button>
    </>}
    title="Launch JupyterLab"
    visible={visible}
    {...props}>
    {showFullConfig?
      <Form form={form}>
        <div className={css.note}
        >
          <Link external path="/docs/reference/command-notebook-config.html">
          Read about notebook settings
          </Link>
        </div><React.Suspense
          fallback={<div className={css.loading}><Spinner className="minHeight" /></div>}>
          <Item
            name="config"
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
              height={400}
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
          slots:1,
          template: 'default',
          type: undefined,
        })}
        labelCol={{ span:8 }}
        onValuesChange={storeConfig}>
        <Item label="Notebook Template" name="template">
          <Select>
            <Option key="default" value="default">Default Task Template</Option>
            {templates.map(temp =>
              <Option key={temp.name} value={temp.name}>{temp.name}</Option>)}
          </Select>
        </Item>
        <Item label="Name" name="name">
          <Input placeholder="Name" />
        </Item>
        <Item
          label="Resource Pool"
          name="pool"
          rules={[ { message: 'Select a resource pool', required: true } ]}>
          <Select
            placeholder="Select a resource pool"
            onChange={handleResourcePoolUpdate}>
            {resourcePools.map(pool =>
              <Option key={pool.name} value={pool.name}>{pool.name}</Option>)}
          </Select>
        </Item>
        <Row justify="end">
          <Col span={8}>
            <Item
              label="Type"
              labelCol={{ span:8 }}
              name="type"
              rules={[ { message: 'Select a resource type', required: true } ]}>
              <RadioGroup
                options={resourceTypeOptions}
                onChange={handleTypeUpdate} />
            </Item>
          </Col>
          <Col span={11}>
            { resourceType === 'GPU'?
              <Item
                label="Number of Slots"
                labelCol={{ span:14 }}
                name="slots"
                rules={[ { message: 'Please choose a number of slots', required: true } ]}>
                <InputNumber min={1} />
              </Item> : null
            }
          </Col>
        </Row>
      </Form>
    }
  </Modal>;
};

export default NotebookModal;
