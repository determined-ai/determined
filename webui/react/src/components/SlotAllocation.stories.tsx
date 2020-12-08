import { withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { ResourceState } from 'types';

import SlotAllocationBar, { Props as ProgressBarProps } from './SlotAllocationBar';

export default {
  component: SlotAllocationBar,
  decorators: [ withKnobs ],
  title: 'SlotAllocationBar',
};

const Wrapper: React.FC<ProgressBarProps> = props => (
  <div style={{ width: 240 }}>
    <SlotAllocationBar {...props} />
  </div>
);

export const Default = (): React.ReactNode => <Wrapper
  resourceStates={[
    ResourceState.Pulling, ResourceState.Running,
  ]}
  totalSlots={4} />;
