import React from 'react';

import { SyncProvider } from 'components/UPlot/SyncProvider';
import { TrialDetails } from 'types';

import SystemMetricChart from './Charts/SystemMetricChart';
import ThroughputMetricChart from './Charts/ThroughputMetricChart';
import TimingMetricChart from './Charts/TimingMetricChart';

export interface Props {
  trial: TrialDetails;
}

const Profiler: React.FC<Props> = ({ trial }) => {
  return (
    <SyncProvider>
      <ThroughputMetricChart trial={trial} />
      <TimingMetricChart trial={trial} />
      <SystemMetricChart trial={trial} />
    </SyncProvider>
  );
};

export default Profiler;
