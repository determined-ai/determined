import React from 'react';

import { LineChart } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import Section from 'components/Section';
import handleError from 'utils/error';

import { ChartProps } from '../types';
import { MetricType } from '../types';
import { useFetchProfilerMetrics } from '../useFetchProfilerMetrics';
import { getScientificNotationTickValues, getUnitForMetricName } from '../utils';

export const TimingMetricChart: React.FC<ChartProps> = ({ trial }) => {
  const timingMetrics = useFetchProfilerMetrics(trial.id, trial.state, MetricType.Timing);

  const yLabel = getUnitForMetricName('seconds');

  return (
    <Section bodyBorder bodyNoPadding title="Timing Metrics">
      <LineChart
        experimentId={trial.id}
        handleError={handleError}
        series={timingMetrics.data}
        xAxis={XAxisDomain.Time}
        xLabel="Time"
        yLabel={yLabel}
        yTickValues={getScientificNotationTickValues}
      />
    </Section>
  );
};

export default TimingMetricChart;
