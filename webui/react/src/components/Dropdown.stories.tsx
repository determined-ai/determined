import { ComponentStory, Meta } from '@storybook/react';
import { Button, Menu } from 'antd';
import React, { useMemo } from 'react';

import AvatarCard from 'shared/components/AvatarCard';
import useUI from 'shared/contexts/stores/UI';
import { useAuth } from 'stores/users';
import { Loadable } from 'utils/loadable';

import Dropdown, { Placement } from './Dropdown';

export default {
  component: Dropdown,
  title: 'Determined/Dropdowns/Dropdown',
} as Meta<typeof Dropdown>;

const content = (
  <Menu
    items={new Array(7).fill(null).map((_, index) => ({ key: index, label: `Menu Item ${index}` }))}
  />
);

export const Default: ComponentStory<typeof Dropdown> = (args) => (
  <Dropdown {...args} content={content}>
    <Button>Default Dropdown</Button>
  </Dropdown>
);

export const Settings: ComponentStory<typeof Dropdown> = (args) => {
  const { user } = Loadable.getOrElse({ checked: true, isAuthenticated: false }, useAuth().auth);
  const { ui } = useUI();
  const menuItems = useMemo(() => {
    return (
      <Menu
        items={[
          { key: 'theme-toggle', label: 'System Mode' },
          {
            key: 'settings',
            label: 'Settings',
          },
          { key: 'sign-out', label: 'Sign Out' },
        ]}
        selectable={false}
      />
    );
  }, []);
  return (
    <Dropdown
      {...args}
      content={menuItems}
      offset={{ x: 16, y: -8 }}
      placement={Placement.BottomLeft}>
      <AvatarCard darkLight={ui.darkLight} displayName={user?.displayName ?? 'Admin'} />
    </Dropdown>
  );
};

Default.args = { placement: Placement.BottomLeft, showArrow: true };
Settings.args = {};
