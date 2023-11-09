import LineChart from 'hew/LineChart';
import React from 'react';

import { Serie, XAxisDomain } from 'types';
import handleError from 'utils/error';
import { capitalizeWord } from 'utils/string';

import { GroupBy } from './ClusterHistoricalUsage.settings';
import css from './ClusterHistoricalUsageChart.module.scss';

interface ClusterHistoricalUsageChartProps {
  chartKey?: number;
  dateRange?: [start: number, end: number];
  formatValues?: (_: uPlot, arg0: number[]) => string[];
  groupBy?: GroupBy;
  height?: number;
  hoursByLabel: Record<string, number[]>;
  label?: string;
  time: string[];
}

const CHART_HEIGHT = 350;

const ClusterHistoricalUsageChart: React.FC<ClusterHistoricalUsageChartProps> = ({
  chartKey,
  dateRange,
  formatValues,
  groupBy,
  height = CHART_HEIGHT,
  hoursByLabel,
  label,
  time,
}: ClusterHistoricalUsageChartProps) => {
  const data: Serie[] = Object.keys(hoursByLabel).map((label) => ({
    data: {
      // convert Unix times from milliseconds to seconds
      [XAxisDomain.Time]: hoursByLabel[label].map((pt, idx) => [Date.parse(time[idx]) / 1000, pt]),
    },
    name: label,
  }));

  return (
    <div className={css.base}>
      <LineChart
        handleError={handleError}
        height={height}
        key={chartKey}
        series={data}
        showLegend
        xAxis={XAxisDomain.Time}
        xLabel={capitalizeWord(groupBy || '')}
        xRange={{
          [XAxisDomain.Time]: dateRange,
          [XAxisDomain.Batches]: undefined,
          [XAxisDomain.Epochs]: undefined,
        }}
        yLabel={label || 'GPU Hours'}
        yTickValues={formatValues}
      />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
