import dayjs from 'dayjs';
import React, { useMemo } from 'react';
import uPlot, { AlignedData, Series } from 'uplot';

import UPlotChart, { Options } from 'components/UPlotChart';
import { glasbeyColor } from 'utils/color';

import { GroupBy } from './ClusterHistoricalUsage';
import css from './ClusterHistoricalUsageChart.module.scss';

interface ClusterHistoricalUsageChartProps {
  groupBy: GroupBy,
  height?: number;
  hoursByLabel: Record<string, number[]>,
  hoursTotal?: number[],
  time: string[],
}

const CHART_HEIGHT = 350;

const ClusterHistoricalUsageChart: React.FC<ClusterHistoricalUsageChartProps> = ({
  groupBy,
  height = CHART_HEIGHT,
  hoursByLabel,
  hoursTotal,
  time,
}: ClusterHistoricalUsageChartProps) => {
  const chartData: AlignedData = useMemo(() => {
    const timeUnix: number[] = time.map(item => Date.parse(item) / 1000);

    const data: AlignedData = [ timeUnix ];
    if (hoursTotal) {
      data.push(hoursTotal);
    }

    Object.keys(hoursByLabel).forEach(label => {
      data.push(hoursByLabel[label]);
    });

    return data;
  }, [ hoursByLabel, hoursTotal, time ]);
  const chartOptions: Options = useMemo(() => {
    let dateFormat = 'MM-DD';
    let timeSeries: Series = { label: 'Day', value: '{YYYY}-{MM}-{DD}' };
    if (groupBy === GroupBy.Month) {
      dateFormat = 'YYYY-MM';
      timeSeries = { label: 'Month', value: '{YYYY}-{MM}' };
    }

    const series: Series[] = [ timeSeries ];
    if (hoursTotal) {
      series.push({
        label: 'total',
        show: false,
        stroke: glasbeyColor(series.length - 1),
        width: 2,
      });
    }
    Object.keys(hoursByLabel).forEach(label => {
      series.push({
        label,
        stroke: glasbeyColor(series.length - 1),
        width: 2,
      });
    });

    return {
      axes: [
        {
          space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
            const rangeSecs = scaleMax - scaleMin;
            const rangeDays = rangeSecs / (24 * 60 * 60);
            const pxPerDay = plotDim / rangeDays;
            return Math.max(
              60,
              pxPerDay * (groupBy === GroupBy.Month ? 28 : 1),
            );
          },
          values: (self, splits) => {
            return splits.map(i => {
              const date = dayjs.utc(i * 1000);
              return date.hour() === 0 ? date.format(dateFormat) : '';
            });
          },
        },
        { label: 'GPU Hours' },
      ],
      height,
      series,
      tzDate: ts => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC'),
    };
  }, [ groupBy, height, hoursByLabel, hoursTotal ]);
  const hasData = useMemo(() => {
    return Object.keys(hoursByLabel)
      .reduce((agg, label) => agg || hoursByLabel[label].length > 0, false);
  }, [ hoursByLabel ]);

  if (!hasData) {
    return (<div>No data to plot.</div>);
  }

  return (
    <div className={css.base}>
      <UPlotChart data={chartData} options={chartOptions} />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
