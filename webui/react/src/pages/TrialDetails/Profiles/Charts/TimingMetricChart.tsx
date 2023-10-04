import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import Section from 'components/Section';
import { ChartProps, MetricType } from 'pages/TrialDetails/Profiles/types';
import { useFetchProfilerMetrics } from 'pages/TrialDetails/Profiles/useFetchProfilerMetrics';
import { getUnitForMetricName } from 'pages/TrialDetails/Profiles/utils';
import { XAxisDomain } from 'types';
import handleError from 'utils/error';

export const TimingMetricChart: React.FC<ChartProps> = ({ trial }) => {
  const timingMetrics = useFetchProfilerMetrics(trial.id, trial.state, MetricType.Timing);

  const yLabel = getUnitForMetricName('seconds');

  return (
    <Section bodyBorder bodyNoPadding title="Timing Metrics">
      <LineChart
        handleError={handleError}
        series={timingMetrics.data}
        showLegend
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={yLabel}
      />
    </Section>
  );
};

export default TimingMetricChart;
