import dayjs from 'dayjs';
import Message from 'determined-ui/Message';
import React, { useMemo } from 'react';

import { LineChart } from 'components/kit/LineChart';

import { GroupBy } from './ClusterHistoricalUsage.settings';
import css from './ClusterHistoricalUsageChart.module.scss';

interface ClusterHistoricalUsageChartProps {
  chartKey?: number;
  dateRange?: [start: number, end: number];
  formatValues?: (self: uPlot, splits: number[]) => string[];
  groupBy?: GroupBy;
  height?: number;
  hoursByLabel: Record<string, number[]>;
  hoursTotal?: number[];
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
  hoursTotal,
  label,
  time,
}: ClusterHistoricalUsageChartProps) => {
  const data: Serie[] = Object.keys(hoursByLabel).map((label) => (
    {
      data: { [XAxisDomain.Time]: hoursByLabel[label].map((pt, idx) =>
        [
          Date.parse(time[idx])/1000,
          pt,
        ]) },
      metricType: '',
      name: label,
    }
  ));

  // const timeUnix: number[] = time.map((item) => Date.parse(item) / 1000);
  // const data: AlignedData = [timeUnix];
  // if (hoursTotal) {
  //   data.push(hoursTotal);
  // }
  // Object.keys(hoursByLabel).forEach((label) => {
  //   data.push(hoursByLabel[label]);
  // });

  const hasData = useMemo(() => {
    return Object.keys(hoursByLabel).reduce(
      (agg, label) => agg || hoursByLabel[label].length > 0,
      false,
    );
  }, [hoursByLabel]);

  // const singlePoint = useMemo(
  //   // one series, and that one series has one point
  //   () => Object.keys(hoursByLabel).length === 1 && Object.values(hoursByLabel)[0].length === 1,
  //   [hoursByLabel],
  // );

  return (
    <div className={css.base}>
      <LineChart series={data} handleError={handleError} xAxis={XAxisDomain.Time} />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
