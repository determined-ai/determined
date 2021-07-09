import React from 'react';

import RouterDecorator from 'storybook/RouterDecorator';
import { ExperimentHyperParamType, MetricType } from 'types';

import HpTrialTable from './HpTrialTable';

export default {
  component: HpTrialTable,
  decorators: [ RouterDecorator ],
  parameters: { layout: 'padded' },
  title: 'HpTrialTable',
};

export const Default = (): React.ReactNode => {
  return <HpTrialTable
    experimentId={1}
    hyperparameters={{ xyz: { type: ExperimentHyperParamType.Categorical, vals: [ true, false ] } }}
    metric={{ name: 'metricA', type: MetricType.Training }}
    trialHps={[
      { hparams: { xyz: true }, id: 1, metric: 0.3 },
      { hparams: { xyz: false }, id: 2, metric: 1.23 },
    ]}
    trialIds={[ 1, 2 ]}
  />;
};
