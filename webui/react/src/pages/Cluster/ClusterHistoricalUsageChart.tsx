import dayjs from 'dayjs';
import Message from 'determined-ui/Message';
import React, { useMemo } from 'react';

import { LineChart } from 'components/kit/LineChart';
import { Serie, XAxisDomain } from 'types';
import handleError from 'utils/error';
import { capitalizeWord } from 'utils/string';

import { GroupBy } from './ClusterHistoricalUsage.settings';
import css from './ClusterHistoricalUsageChart.module.scss';

interface ClusterHistoricalUsageChartProps {
  formatValues?: (_: uPlot, arg0: number[]) => string[];
  groupBy?: GroupBy;
  height?: number;
  hoursByLabel: Record<string, number[]>;
  label?: string;
  time: string[];
}

const CHART_HEIGHT = 350;

const ClusterHistoricalUsageChart: React.FC<ClusterHistoricalUsageChartProps> = ({
  formatValues,
  groupBy,
  height = CHART_HEIGHT,
  hoursByLabel,
  label,
  time,
}: ClusterHistoricalUsageChartProps) => {
  const data: Serie[] = Object.keys(hoursByLabel).map((label) => ({
    data: {
      [XAxisDomain.Time]: hoursByLabel[label].map((pt, idx) => [Date.parse(time[idx]) / 1000, pt]),
    },
    metricType: '',
    name: label,
  }));

  return (
    <div className={css.base}>
      <LineChart
        handleError={handleError}
        height={height}
        series={data}
        xAxis={XAxisDomain.Time}
        xLabel={capitalizeWord(groupBy || '')}
        yLabel={label || 'GPU Hours'}
        yTickValues={formatValues}
      />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
