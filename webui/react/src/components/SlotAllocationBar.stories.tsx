import { withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { ShirtSize } from 'themes';
import { ResourceState } from 'types';

import SlotAllocationBar, { Props as SlotAllocationProps } from './SlotAllocationBar';

export default {
  component: SlotAllocationBar,
  decorators: [ withKnobs ],
  title: 'SlotAllocationBar',
};

const Wrapper: React.FC<SlotAllocationProps> = props => (
  <div style={{ minWidth: 500 }}>
    <SlotAllocationBar {...props} />
  </div>
);

export const Default = (): React.ReactNode => <Wrapper
  resourceStates={[
    ResourceState.Pulling, ResourceState.Running,
  ]}
  showLegends
  size={ShirtSize.big}
  totalSlots={4} />;

export const NoLegends = (): React.ReactNode => <Wrapper
  resourceStates={[
    ResourceState.Pulling, ResourceState.Running,
  ]}
  size={ShirtSize.big}
  totalSlots={4} />;

export const ZeroTotalSlots = (): React.ReactNode => <Wrapper
  resourceStates={[]}
  showLegends
  size={ShirtSize.big}
  totalSlots={0} />;

export const ZeroTotalSlotsWithMismatchingProps = (): React.ReactNode => <Wrapper
  resourceStates={[ ResourceState.Running ]}
  showLegends
  size={ShirtSize.big}
  totalSlots={0} />;
