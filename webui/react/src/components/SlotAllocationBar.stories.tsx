import { number, select, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { enumToOptions } from 'storybook/utils';
import { CommandState, RunState } from 'types';

import SlotAllocationBar, { Props } from './SlotAllocationBar';

export default {
  component: SlotAllocationBar,
  decorators: [ withKnobs ],
  title: 'SlotAllocationBar',
};

const Wrapper: React.FC<Props> = props => (
  <div style={{ width: 240 }}>
    <SlotAllocationBar {...props} />
  </div>
);

const cmdStateOptions = enumToOptions<CommandState>(CommandState);
const runStateOptions = enumToOptions<RunState>(RunState);

export const Default = (): React.ReactNode => <Wrapper parts={[
  {
    color: 'red',
    label: 'labelA',
    percent: 0.3,
  },
]} />;
// export const Full = (): React.ReactNode => <Wrapper percent={100} state={RunState.Completed} />;
// export const Empty = (): React.ReactNode => <Wrapper percent={0} state={RunState.Paused} />;

// export const Custom = (): React.ReactNode => <Wrapper
//   percent={number('Percent', 50, { max: 100, min: 0 })}
//   state={select<RunState | CommandState>(
//     'State',
//     { ...cmdStateOptions, ...runStateOptions },
//     RunState.Active,
//   )}
// />;
