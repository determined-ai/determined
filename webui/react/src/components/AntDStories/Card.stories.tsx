import { ComponentStory, Meta } from '@storybook/react';
import { Card } from 'antd';
import React from 'react';

import loremIpsum from 'shared/utils/loremIpsum';

export default {
  argTypes: { size: { control: { options: ['small', 'default'], type: 'inline-radio' } } },
  component: Card,
  title: 'Ant Design/Card',
} as Meta<typeof Card>;

export const Default: ComponentStory<typeof Card> = (args) => <Card {...args}>{loremIpsum}</Card>;

Default.args = {
  extra: '',
  hoverable: false,
  size: 'small',
  title: 'Title',
};
