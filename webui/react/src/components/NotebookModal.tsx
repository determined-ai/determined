import { Button, Col, InputNumber, Modal, Row } from 'antd';
import { Form, Input, Select } from 'antd';
import { ModalProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';

import { getResourcePools, getTemplates } from 'services/api';
import { ResourcePool, Template } from 'types';
import { launchNotebook } from 'utils/task';

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
  const [ fullConfig, setFullConfig ] = useState('');
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
    if(showFullConfig) {
      null; //call api to generate configuration
      //setFullConfig(value)
    }
  }, [ showFullConfig ]);

  const handleConfigChange = useCallback((value) => {
    setFullConfig(value);
  },[]);

  const handleSecondary = useCallback(() => {
    setShowFullConfig(show => !show);
  },[]);

  const handleCreateEnvironment = useCallback(
    (values) =>{
      if (showFullConfig) {
        //launchNotebook with full config
      } else {
        if(values.template !== '') {
          launchNotebook(values.resourceType === 'GPU'? values.slots : 0, values.template);
        } else {
          launchNotebook(values.resourceType === 'GPU'? values.slots : 0);
        }
      }
    },
    [ ],
  );

  const handleResourcePoolUpdate = useCallback((e) => {
    e;
    /*
    if (e === '') {
      setResourceTypeOptions([ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]);
    } else {
      const pool = resourcePools.find(p => p.name === e);
      if (pool){
        const options = [];
        if (pool.cpuContainerCapacityPerAgent > 0) {
          options.push({ id:'CPU', label:'CPU' });
        }
        if (pool.slotsPerAgent > 0) {
          options.push({ id:'GPU', label:'GPU' });
        }
        setResourceTypeOptions(options)
      }
    }
    */

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
      <>
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
        <MonacoEditor
          height={400}
          language='yaml'
          value={fullConfig}
          onChange={handleConfigChange} />
      </> :
      <Form form={form} initialValues={{ slots:1 }} labelCol={{ span:8 }}>
        <Row justify='end'>
          <Item
            label='Type'
            labelCol={{ span:8 }}
            name='type'
            rules={[ { message: 'Please choose a resource type', required: true } ]}>
            <RadioGroup
              options={resourceTypeOptions}
              onChange={(e) => handleTypeUpdate(e)} />
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
        <Item label='Notebook Template' name='template'>
          <Dropdown options={templates.map(template => template.name)} />
        </Item>
        <Item label='Name' name="name">
          <Input placeholder='Name' />
        </Item>
        <Item label='Resource Pool' name="pool">
          <Dropdown
            options={resourcePools.map(pool => pool.name)}
            onChange={handleResourcePoolUpdate} />
        </Item>
      </Form>
    }
  </Modal>;
};

interface DropdownProps {
  onChange?: (e: string) => void;
  options?: string[];
}

const Dropdown: React.FC<DropdownProps> = ({ options, onChange }: DropdownProps) => {
  return options? <Select defaultValue='' style={{ minWidth:120 }} onChange={onChange}>
    <Option key='empty' value=''>---Empty---</Option>
    {options.map(option => <Option key={option} value={option}>{option}</Option>)}
  </Select> : null;
};

export default NotebookModal;
