import React from 'react';

import { TrialDetails } from 'types';

import SystemMetricChart from './Charts/SystemMetricChart';
import ThroughputMetricChart from './Charts/ThroughputMetricChart';
import TimingMetricChart from './Charts/TimingMetricChart';
// import css from './Profiler.module.scss';

export interface Props {
  trial: TrialDetails;
}

const Profiler: React.FC<Props> = ({ trial }) => {
  return (
    <div>
      <ThroughputMetricChart trial={trial} />
      <TimingMetricChart trial={trial} />
      <SystemMetricChart trial={trial} />
    </div>
  );
};

export default Profiler;
