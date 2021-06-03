import { Alert } from 'antd';
import React, { useMemo } from 'react';
import uPlot, { AlignedData } from 'uplot';

import Section from 'components/Section';
import UPlotChart, { Options } from 'components/UPlotChart';
import { CHART_HEIGHT } from 'pages/TrialDetails/TrialDetailsProfiles';
import { TrialDetails } from 'types';
import { glasbeyColor } from 'utils/color';
import { findFactorOfNumber } from 'utils/number';

import { convertMetricsToUplotData, MetricType, useFetchMetrics } from './utils';

export interface Props {
  trial: TrialDetails;
}

const TimingMetricChart: React.FC<Props> = ({ trial }: Props) => {
  const timingMetrics = useFetchMetrics(trial.id, MetricType.Timing);

  const chartData: AlignedData = useMemo(() => {
    return convertMetricsToUplotData(timingMetrics.dataByBatch, timingMetrics.names);
  }, [ timingMetrics ]);
  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        {
          label: 'Batch',
          space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
            const range = scaleMax - scaleMin + 1;
            const factor = findFactorOfNumber(range).reverse()
              .find(factor => plotDim / factor > 35);
            return factor ? Math.min(70, (plotDim / factor)) : 35;
          },
        },
        { label: 'Milliseconds' },
      ],
      height: CHART_HEIGHT,
      scales: { x: { time: false } },
      series: [
        { label: 'Batch' },
        ...timingMetrics.names.map((name, index) => ({
          label: name,
          points: { show: false },
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
      tzDate: ts => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC'),
    };
  }, [ timingMetrics.names ]);

  if (timingMetrics.isEmpty) {
    return (
      <Section title="Timing Metrics">
        <Alert
          description="Timing metrics may not be available for your framework."
          message="No data found."
          type="warning"
        />
      </Section>
    );
  }

  return (
    <Section bodyBorder title="Timing Metrics">
      <UPlotChart data={chartData} options={chartOptions} />
    </Section>
  );
};

export default TimingMetricChart;
