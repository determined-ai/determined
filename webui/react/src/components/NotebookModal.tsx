import { Button, Col, InputNumber, Modal, Row } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { getResourcePools, getTemplates } from 'services/api';
import { RawJson, ResourcePool, Template } from 'types';
import { launchNotebook, previewNotebook } from 'utils/task';

import Link from './Link';
import RadioGroup from './RadioGroup';

const { Option } = Select;
const { Item } = Form;

interface Props extends ModalProps {
  visible?: boolean;
}

/*
Proposed WebUI flow for notebook creation.
User clicks “Launch Notebooks” from the navigation bar.
A modal opens up with a form with the following items:
Link to the documentation
Dropdown for notebook templates.
Text box for name.
Dropdown for resource pool. <optional>
Radio button for CPU-only (0 slot) or GPU slots (will depend on resource pool selection)
cpuContainerCapacityPerAgent > 0 if resource pool has cpu capacity
slotsPerAgent > 0 if resource pool has GPU capacity
Text box for number of slots. (will only show if GPU is selected)
Primary button with the label “Create Notebook Environment”
Secondary button with the label “Edit full config”.
If the user clicks on “Edit full config”, the content area of the modal
switches to a view with the following items: (this action will require an API call to
  interpolate the user selected values into the config to populate the config editor)
Link to the documentation.
Editor with the full content populated with the name, resource pool, number of slots, and template.
Primary button with the label “Create Notebook Environment”
Secondary button with a back arrow.
*/

const NotebookModal: React.FC<Props> = (
  { visible = false, ...props }: Props,
) => {
  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ templates, setTemplates ] = useState<Template[]>([]);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ resourceTypeOptions, setResourceTypeOptions ] =
    useState<{id:string, label:string}[]>([ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]);
  const [ resourceType, setResourceType ] = useState(undefined);
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
      const pool = resourcePools.find(p => p.name === e);
      if (pool){
        const options = [];
        if (pool.cpuContainerCapacityPerAgent > 0) {
          options.push({ id:'CPU', label:'CPU' });
        }
        if (pool.slotsPerAgent && pool.slotsPerAgent > 0) {
          options.push({ id:'GPU', label:'GPU' });
        }
        setResourceTypeOptions(options);
      }
    }

    //Type form field should ONLY appear if both slotsPerAgent and
    //cpuCapacityPerAgent are both greater than 0

  },[ ]);

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
      <Form form={form} initialValues={{ pool: '', slots:1, template: '' }} labelCol={{ span:8 }}>
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
          <Item
            label='Type'
            labelCol={{ span:8 }}
            name='type'
            rules={[ { message: 'Choose a resource type', required: true } ]}>
            <RadioGroup
              options={resourceTypeOptions}
              onChange={handleTypeUpdate} />
          </Item>
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
