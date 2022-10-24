import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { ShirtSize } from 'themes';
import { ResourceState } from 'types';

import SlotAllocationBar from './SlotAllocationBar';

export default {
  argTypes: {
    resourceStates: { control: { options: ResourceState, type: 'inline-check' } },
    size: { control: { options: ShirtSize, type: 'select' } },
  },
  component: SlotAllocationBar,
  title: 'Determined/Bars/SlotAllocationBar',
} as Meta<typeof SlotAllocationBar>;

export const Default: ComponentStory<typeof SlotAllocationBar> = (args) => (
  <div style={{ minWidth: 500 }}>
    <SlotAllocationBar {...args} />
  </div>
);

Default.args = {
  resourceStates: [ResourceState.Pulling, ResourceState.Running],
  showLegends: true,
  size: ShirtSize.Large,
  totalSlots: 4,
};
