import { Button, Col, InputNumber, Modal, Row } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import useStorage from 'hooks/useStorage';
import { getResourcePools, getTemplates } from 'services/api';
import { RawJson, ResourcePool, Template } from 'types';
import { launchNotebook, previewNotebook } from 'utils/task';

import Link from './Link';
import RadioGroup from './RadioGroup';

const { Option } = Select;
const { Item } = Form;

const STORAGE_PATH = 'notebook-launch';
const STORAGE_KEY = 'notebook-config';

interface Props extends ModalProps {
  visible?: boolean;
}

const NotebookModal: React.FC<Props> = (
  { visible = false, ...props }: Props,
) => {
  const storage = useStorage(STORAGE_PATH);
  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ templates, setTemplates ] = useState<Template[]>([]);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ resourceTypeOptions, setResourceTypeOptions ] =
    useState<{id:string, label:string}[]>([ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]);
  const [ resourceType, setResourceType ] =
    useState(storage.getWithDefault(STORAGE_KEY, { type: undefined }).type);
  const [ form ] = Form.useForm();

  useEffect(() => {
    const fetchTemplates = async () => {
      setTemplates(await getTemplates({}));
    };
    fetchTemplates();
  }, []);

  useEffect(() => {
    const fetchResourcePools = async () => {
      setResourcePools( await getResourcePools({}));
    };
    fetchResourcePools();
  }, []);

  useEffect(()=> {
    const fetchConfig = async (values: RawJson) => {
      if(showFullConfig) {
        const config = await previewNotebook(
          values.slots,
          values.template,
          values.name,
          values.pool,
        );
        form.setFieldsValue({ config: yaml.dump(config) });
      }
    };
    fetchConfig(form.getFieldsValue(true));
  }, [ showFullConfig, form ]);

  const storeConfig = useCallback((_, values) => {
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
          values.resourceType === 'GPU'? values.slots : 0,
          values.template === ''? undefined : values.template,
          values.name,
          values.pool,
        );
      }
    },
    [ showFullConfig, form ],
  );

  const handleResourcePoolUpdate = useCallback((e) => {
    if (e === '') {
      setResourceTypeOptions([ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]);
    } else {
      const pool = resourcePools.find(pool => pool.name === e);
      if (pool){
        const options = [];
        if (pool.cpuContainerCapacityPerAgent > 0) {
          options.push({ id:'CPU', label:'CPU' });
        }
        if (pool.slotsPerAgent && pool.slotsPerAgent > 0) {
          options.push({ id:'GPU', label:'GPU' });
        }
        setResourceTypeOptions(options);
        form.setFieldsValue({ type:undefined });
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
    title='Launch JupyterLab'
    visible={visible}
    {...props}>
    {showFullConfig?
      <Form form={form}>
        <div style={{
          backgroundColor:'rgb(230,230,230)',
          border:'1px solid rgb(200,200,200)',
          marginBottom: '4px',
          padding: 2,
          textAlign:'center',
        }}
        >
          <Link external path="/docs/reference/command-notebook-config.html">
          Read about notebook settings
          </Link>
        </div>
        <Item
          name='config'
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
            language='yaml'
            options={{
              minimap: { enabled: false },
              scrollBeyondLastLine: false,
              selectOnLineNumbers: true,
            }} />
        </Item>
      </Form> :
      <Form
        form={form}
        initialValues={storage.getWithDefault(STORAGE_KEY, {
          pool: '',
          slots:1,
          template: '',
          type: undefined,
        })}
        labelCol={{ span:8 }}
        onValuesChange={storeConfig}>
        <Item label='Notebook Template' name='template'>
          <Select
            style={{ minWidth:120 }}>
            <Option key='' value=''>---Empty---</Option>
            {templates.map(temp =>
              <Option key={temp.name} value={temp.name}>{temp.name}</Option>)}
          </Select>
        </Item>
        <Item label='Name' name="name">
          <Input placeholder='Name' />
        </Item>
        <Item label='Resource Pool' name="pool">
          <Select
            style={{ minWidth:120 }}
            onChange={handleResourcePoolUpdate}>
            <Option key='' value=''>---Empty---</Option>
            {resourcePools.map(pool =>
              <Option key={pool.name} value={pool.name}>{pool.name}</Option>)}
          </Select>
        </Item>
        <Row justify='end'>
          <Col span={8}>
            <Item
              label='Type'
              labelCol={{ span:8 }}
              name='type'
              rules={[ { message: 'Choose a resource type', required: true } ]}>
              <RadioGroup
                options={resourceTypeOptions}
                onChange={handleTypeUpdate} />
            </Item>
          </Col>
          <Col span={11}>
            { resourceType === 'GPU'?
              <Item
                label='Number of Slots'
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
