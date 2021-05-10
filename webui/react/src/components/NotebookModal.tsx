import { Modal } from 'antd';
import { Form, Input, Select } from 'antd';
import React, { useCallback, useState } from 'react';

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
  const [ form ] = Form.useForm();

  const handleSecondary = useCallback(() => {
    setShowFullConfig(show => !show);
  },[]);

  const handleCreateEnvironment = useCallback(() => {
    null;
  },[]);

  const handleNameUpdate = useCallback((e) => {
    form.setFieldsValue({ name: e });
  },[]);

  const handlTemplateUpdate = useCallback((e) => {
    form.setFieldsValue({ template: e });
  },[]);

  const handleResourcePoolUpdate = useCallback((e) => {
    form.setFieldsValue({ pool: e });
  },[]);

  const handleTypeUpdate = useCallback((e) => {
    form.setFieldsValue({ type: e });
  },[]);

  return <Modal
    cancelButtonProps={{ onClick: handleSecondary }}
    cancelText={showFullConfig ? 'Back' : 'Edit Full Config'}
    okButtonProps={{ onClick: handleCreateEnvironment }}
    okText='Create Notebook Environment'
    title='Notebook Settings'
    visible={forceVisible}>
    {showFullConfig? <Input.TextArea defaultValue='' /> :
      <>
        <Link><a href=''>Documentation</a></Link>
        <Form labelAlign='left' labelCol={{ span:8 }}>
          <Item label='Notebook Template'>
            <Dropdown options={[]} onChange={handlTemplateUpdate} />
          </Item>
          <Item label='Name'><Input placeholder='Name' onChange={handleNameUpdate} /></Item>
          <Item label='Resource Pool'>
            <Dropdown options={[]} onChange={handleResourcePoolUpdate} />
          </Item>
          <Item label='Type'>
            <RadioGroup
              options={[ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]}
              onChange={(e) => handleTypeUpdate(e)} />
          </Item>
          <Item label='Number of Slots'><Input defaultValue={0} type='number' /></Item>
        </Form>
      </>
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
