import LineChart from 'determined-ui/LineChart';
import React, { useMemo } from 'react';

import { Serie, XAxisDomain } from 'types';
import handleError from 'utils/error';
import { capitalizeWord } from 'utils/string';

import { GroupBy } from './ClusterHistoricalUsage.settings';
import css from './ClusterHistoricalUsageChart.module.scss';

interface ClusterHistoricalUsageChartProps {
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
  dateRange,
  formatValues,
  groupBy,
  height = CHART_HEIGHT,
  hoursByLabel,
  label,
  time,
}: ClusterHistoricalUsageChartProps) => {
  const singlePoint = useMemo(
    // one series, and that one series has one point
    () => Object.keys(hoursByLabel).length === 1 && Object.values(hoursByLabel)[0].length === 1,
    [hoursByLabel],
  );

  const data: Serie[] = Object.keys(hoursByLabel).map((label) => ({
    data: {
      [XAxisDomain.Time]: hoursByLabel[label].map((pt, idx) => [Date.parse(time[idx]) / 1000, pt]),
    },
    name: label,
  }));

  const adjustedDateRange: [number, number] | undefined = useMemo(() => {
    return (
      dateRange ??
      (singlePoint
        ? [
            new Date(`${new Date().getFullYear()}-01-01`).getTime() / 1000,
            new Date(`${new Date().getFullYear() + 1}-01-01`).getTime() / 1000,
          ]
        : undefined)
    );
  }, [singlePoint, dateRange]);

  return (
    <div className={css.base}>
      <LineChart
        handleError={handleError}
        height={height}
        series={data}
        xAxis={XAxisDomain.Time}
        xLabel={capitalizeWord(groupBy || '')}
        xRange={{
          [XAxisDomain.Time]: adjustedDateRange,
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
