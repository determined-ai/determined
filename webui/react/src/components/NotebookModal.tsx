import { Modal } from 'antd';
import { Form, Input, Select } from 'antd';
import React, { useCallback } from 'react';

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

  const handleEditConfig = useCallback(() => {
    'placeholder';
  },[]);

  const handleCreateEnvironment = useCallback(() => {
    'placeholder';
  },[]);

  return <Modal
    cancelButtonProps={{ onClick: handleEditConfig }}
    cancelText='Edit Full Config'
    okButtonProps={{ onClick: handleCreateEnvironment }}
    okText='Create Notebook Environment'
    title='Notebook Settings'
    visible={forceVisible}>
    <Link><a href=''>Documentation</a></Link>
    <Form labelAlign='left' labelCol={{ span:8 }}>
      <Item label='Notebook Template'><Dropdown options={[]} /></Item>
      <Item label='Name'><Input placeholder='Name' /></Item>
      <Item label='Resource Pool'><Dropdown options={[]} /></Item>
      <Item label='Type'>
        <RadioGroup options={[ { id:'CPU', label:'CPU' }, { id:'GPU', label:'GPU' } ]} />
      </Item>
      <Item label='Number of Slots'><Input defaultValue={0} type='number' /></Item>
    </Form>
  </Modal>;
};

interface DropdownProps {
  options?: string[]
}

const Dropdown: React.FC<DropdownProps> = ({ options }: DropdownProps) => {
  return options? <Select style={{ minWidth:120 }}>
    {options.map(option => <Option key={option} value={option}>{option}</Option>)}
  </Select> : null;
};

export default NotebookModal;
