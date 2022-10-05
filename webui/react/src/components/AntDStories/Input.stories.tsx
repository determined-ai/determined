import { ComponentStory, Meta } from '@storybook/react';
import { Input } from 'antd';
import React from 'react';

export default {
  argTypes: {
    maxLength: { control: { type: 'number' } },
    size: { control: { options: ['small', 'middle', 'large'], type: 'inline-radio' } },
  },
  component: Input,
  title: 'Ant Design/Input',
} as Meta<typeof Input>;

export const Default: ComponentStory<typeof Input> = (args) => <Input {...args} />;

Default.args = {
  placeholder: 'Placeholder text',
  prefix: '',
  showCount: false,
  size: 'middle',
  suffix: '',
};

export const TextArea: ComponentStory<typeof Input.TextArea> = (args) => (
  <Input.TextArea {...args} />
);

TextArea.args = {
  allowClear: false,
  autoSize: false,
  bordered: true,
  placeholder: 'Placeholder text',
  showCount: false,
  size: 'middle',
};

export const Search: ComponentStory<typeof Input.Search> = (args) => <Input.Search {...args} />;

Search.args = {
  enterButton: false,
  loading: false,
  placeholder: 'Placeholder text',
  prefix: '',
  showCount: false,
  size: 'middle',
  suffix: '',
};

export const Password: ComponentStory<typeof Input.Password> = (args) => (
  <Input.Password {...args} />
);

Password.args = {
  placeholder: 'Placeholder text',
  prefix: '',
  showCount: false,
  size: 'middle',
  visibilityToggle: true,
};
