import React, { useMemo } from 'react';

import { Serie, XAxisDomain } from 'components/kit/internal/types';
import { LineChart } from 'components/kit/LineChart';
import Message from 'components/kit/Message';
import handleError from 'utils/error';

import { GroupBy } from './ClusterHistoricalUsage.settings';
import css from './ClusterHistoricalUsageChart.module.scss';

interface ClusterHistoricalUsageChartProps {
  groupId?: string;
  dateRange?: [start: number, end: number];
  formatYvalue?: (val: number) => string;
  groupBy?: GroupBy;
  hoursByLabel: Record<string, number[]>;
  hoursTotal?: number[];
  label?: string;
  time: string[];
}

const ClusterHistoricalUsageChart: React.FC<ClusterHistoricalUsageChartProps> = ({
  groupId,
  dateRange,
  formatYvalue,
  groupBy,
  hoursByLabel,
  hoursTotal,
  label,
  time,
}: ClusterHistoricalUsageChartProps) => {
  const chartData: Serie[] = useMemo(() => {
    const timeUnix: number[] = time.map((item) => Date.parse(item) / 1000);
    const series: Serie[] = [];

    if (hoursTotal) {
      series.push({
        data: { [XAxisDomain.Time]: hoursTotal?.map((val, i) => [timeUnix[i], val]) },
        name: 'total',
      });
    }

    series.push(
      ...Object.keys(hoursByLabel).map((label): Serie => {
        return {
          data: {
            [XAxisDomain.Time]: hoursByLabel[label].map((yValue, i): [x: number, y: number] => {
              return [timeUnix[i], yValue];
            }),
          },
          name: label,
        };
      }),
    );
    return series;
  }, [hoursByLabel, hoursTotal, time]);

  const hasData = useMemo(() => {
    return Object.keys(hoursByLabel).reduce(
      (agg, label) => agg || hoursByLabel[label].length > 0,
      false,
    );
  }, [hoursByLabel]);

  if (!hasData) {
    return <Message icon="warning" title="No data to plot." />;
  }

  return (
    <div className={css.base}>
      <LineChart
        group={groupId}
        handleError={handleError}
        series={chartData}
        showLegend
        xAxis={XAxisDomain.Time}
        xLabel={groupBy === GroupBy.Month ? 'Month' : 'Day'}
        xValueRange={dateRange}
        yLabel={label}
        yValueFormatter={formatYvalue}
      />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
