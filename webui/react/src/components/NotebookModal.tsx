import { Button, Modal } from 'antd';
import { Form, Input, Select } from 'antd';
//eslint-disable-next-line
import React, { useCallback, useEffect, useState } from 'react';
import { getResourcePools } from 'services/api';
import { ResourcePool } from 'types';

import Link from './Link';
import RadioGroup from './RadioGroup';

const { Option } = Select;
const { Item } = Form;

interface Props {
  forceVisible?: boolean;
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
  { forceVisible = false }: Props,
) => {
  const [ showFullConfig, setShowFullConfig ] = useState(false);
  const [ resourcePools, setResourcePools ] = useState<ResourcePool[]>([]);
  const [ resourceType, setResourceType ] = useState(undefined);
  const [ form ] = Form.useForm();

  useEffect(() => {
    null; //get templates from api
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
    }
  }, [ showFullConfig ]);

  const handleSecondary = useCallback(() => {
    setShowFullConfig(show => !show);
  },[]);

  const handleCreateEnvironment = useCallback(() => {
    null; //call api to create environment
  },[]);

  const handleNameUpdate = useCallback((e) => {
    e;
  },[ ]);

  const handlTemplateUpdate = useCallback((e) => {
    e;
  },[ ]);

  const handleResourcePoolUpdate = useCallback((e) => {
    e; //get slots per agent and cpuCapacityPerAgent

    //Type form field should ONLY appear if both slotsPerAgent and
    //cpuCapacityPerAgent are both greater than 0
  },[ ]);

  const handleTypeUpdate = useCallback((e) => {
    setResourceType(e);
  },[]);

  return <Modal
    footer={<>
      <Button onClick={handleSecondary}>{showFullConfig ? 'Back' : 'Edit Full Config'}</Button>
      <Button type="primary" onClick={handleCreateEnvironment}>Create Notebook Environment</Button>
    </>}
    title='Notebook Settings'
    visible={forceVisible}>
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
          <Link path="/docs/reference/command-notebook-config.html">
          Read about notebook settings
          </Link>
        </div>

        <Input.TextArea defaultValue='' />
      </> :
      <Form form={form} labelCol={{ span:8 }}>
        <Item label='Notebook Template'>
          <Dropdown options={[]} onChange={handlTemplateUpdate} />
        </Item>
        <Item label='Name' name="name" required>
          <Input placeholder='Name' onChange={handleNameUpdate} />
        </Item>
        <Item label='Resource Pool' name="pool">
          <Dropdown
            options={resourcePools.map(pool => pool.name)}
            onChange={handleResourcePoolUpdate} />
        </Item>
        <Item label='Type' name='type' required>
          <RadioGroup
            options={[ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]}
            onChange={(e) => handleTypeUpdate(e)} />
        </Item>
        { resourceType === 'GPU'?
          <Item label='Number of Slots' name="slots" required>
            <Input defaultValue={1} type='number' />
          </Item> : null
        }
      </Form>
    }
  </Modal>;
};

interface DropdownProps {
  onChange: (e: string) => void;
  options?: string[];
}

const Dropdown: React.FC<DropdownProps> = ({ options, onChange }: DropdownProps) => {
  return options? <Select style={{ minWidth:120 }} onChange={onChange}>
    {options.map(option => <Option key={option} value={option}>{option}</Option>)}
  </Select> : null;
};

export default NotebookModal;
