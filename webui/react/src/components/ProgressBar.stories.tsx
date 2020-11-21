import { number, select, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { enumToOptions } from 'storybook/utils';
import { CommandState, RunState } from 'types';

import ProgressBar, { Props as ProgressBarProps } from './ProgressBar';

export default {
  component: ProgressBar,
  decorators: [ withKnobs ],
  title: 'ProgressBar',
};

const Wrapper: React.FC<ProgressBarProps> = props => (
  <div style={{ width: 240 }}>
    <ProgressBar {...props} />
  </div>
);

const cmdStateOptions = enumToOptions<CommandState>(CommandState);
const runStateOptions = enumToOptions<RunState>(RunState);

export const Default = (): React.ReactNode => <Wrapper percent={53} state={RunState.Active} />;
export const Full = (): React.ReactNode => <Wrapper percent={100} state={RunState.Completed} />;
export const Empty = (): React.ReactNode => <Wrapper percent={0} state={RunState.Paused} />;

export const Custom = (): React.ReactNode => <Wrapper
  percent={number('Percent', 50, { max: 100, min: 0 })}
  state={select<RunState | CommandState>(
    'State',
    { ...cmdStateOptions, ...runStateOptions },
    RunState.Active,
  )}
/>;
