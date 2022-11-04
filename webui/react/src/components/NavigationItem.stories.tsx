import { Meta, Story } from '@storybook/react';
import { Button } from 'antd';
import React from 'react';

import Icon from 'shared/components/Icon';

import { NavigationItem } from './NavigationSideBar';
import css from './NavigationSideBar.module.scss';

export default {
  argTypes: {
    action: { control: { type: 'boolean' } },
    icon: {
      control: {
        options: [
          'arrow-down',
          'arrow-up',
          'dai-logo',
          'cluster',
          'collapse',
          'command',
          'expand',
          'experiment',
          'grid',
          'jupyter-lab',
          'list',
          'lock',
          'notebook',
          'overflow-horizontal',
          'overflow-vertical',
          'shell',
          'star',
          'tensor-board',
          'tensorflow',
          'user',
          'user-small',
        ],
        type: 'select',
      },
    },
  },
  component: NavigationItem,
  title: 'Determined/Buttons/NavigationItem',
} as Meta<typeof NavigationItem>;

type NavigationItemProps = React.ComponentProps<typeof NavigationItem>;

export const Default: Story<NavigationItemProps & { action: boolean }> = ({ action, ...args }) => (
  <div className={css.base}>
    <NavigationItem
      action={
        action ? (
          <Button type="text">
            <Icon name="add-small" size="tiny" />
          </Button>
        ) : undefined
      }
      {...args}
    />
  </div>
);

Default.args = {
  action: false,
  icon: 'experiment',
  label: 'Navigation Button',
};
