import { Modal } from 'antd';
import { Select } from 'antd';
import React, { } from 'react';

const { Option } = Select;

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
  return <Modal visible={forceVisible}>
    <p>test</p>
    <LabelledLine content={<NotebookTemplates />} label='Notebook Template' />
  </Modal>;
};

const NotebookTemplates: React.FC = () => {
  return <Select style={{ minWidth:120 }}>
    <Option key='placeholder' value='placeholder'>placeholder</Option>
  </Select>;
};

interface LabelledLineProps {
  content?: JSX.Element;
  label: string;
}

const LabelledLine: React.FC<LabelledLineProps> = ({ content, label }: LabelledLineProps) => {
  return content ?
    <div style={{ alignItems: 'center', display:'flex', justifyContent:'space-between' }}>
      <label>{label}</label>
      {content}
    </div> : null;
};

export default NotebookModal;
