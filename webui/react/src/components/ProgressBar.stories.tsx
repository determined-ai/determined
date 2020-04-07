import React from 'react';

import { RunState } from 'types';

import ProgressBar from './ProgressBar';

export default {
  component: ProgressBar,
  title: 'ProgressBar',
};

export const Default = (): React.ReactNode => <ProgressBar percent={53} state={RunState.Active} />;
export const Full = (): React.ReactNode => <ProgressBar percent={100} state={RunState.Completed} />;
export const Empty = (): React.ReactNode => <ProgressBar percent={0} state={RunState.Paused} />;
