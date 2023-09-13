import dayjs from 'dayjs';
import React, { useMemo } from 'react';
import uPlot, { AlignedData, Series } from 'uplot';

import Message, { MessageType } from 'components/Message';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import { glasbeyColor } from 'utils/color';

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
  const chartData: AlignedData = useMemo(() => {
    const timeUnix: number[] = time.map((item) => Date.parse(item) / 1000);

    const data: AlignedData = [timeUnix];
    if (hoursTotal) {
      data.push(hoursTotal);
    }

    Object.keys(hoursByLabel).forEach((label) => {
      data.push(hoursByLabel[label]);
    });

    return data;
  }, [hoursByLabel, hoursTotal, time]);

  const hasData = useMemo(() => {
    return Object.keys(hoursByLabel).reduce(
      (agg, label) => agg || hoursByLabel[label].length > 0,
      false,
    );
  }, [hoursByLabel]);

  const singlePoint = useMemo(
    // one series, and that one series has one point
    () => Object.keys(hoursByLabel).length === 1 && Object.values(hoursByLabel)[0].length === 1,
    [hoursByLabel],
  );

  const chartOptions: Options = useMemo(() => {
    let dateFormat = 'MM-DD';
    let timeSeries: Series = { label: 'Day', value: '{YYYY}-{MM}-{DD}' };
    if (groupBy === GroupBy.Month) {
      dateFormat = 'YYYY-MM';
      timeSeries = { label: 'Month', value: '{YYYY}-{MM}' };
    }

    const series: Series[] = [timeSeries];
    if (hoursTotal) {
      series.push({
        label: 'total',
        show: false,
        stroke: glasbeyColor(series.length - 1),
        width: 2,
      });
    }
    Object.keys(hoursByLabel).forEach((label) => {
      series.push({
        label,
        stroke: glasbeyColor(series.length - 1),
        width: 2,
      });
    });

    return {
      axes: [
        {
          label: timeSeries.label,
          space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
            const rangeSecs = scaleMax - scaleMin;
            const rangeDays = rangeSecs / (24 * 60 * 60);
            const pxPerDay = plotDim / rangeDays;
            return Math.max(60, pxPerDay * (groupBy === GroupBy.Month ? 28 : 1));
          },
          values: (self, splits) => {
            return splits.map((i) => {
              const date = dayjs.utc(i * 1000);
              return date.hour() === 0 ? date.format(dateFormat) : '';
            });
          },
        },
        { label: label ? label : 'GPU Hours', values: formatValues },
      ],
      height,
      key: chartKey,
      scales: {
        x: {
          auto: !singlePoint,
          range:
            dateRange ??
            (singlePoint
              ? [
                  new Date(`${new Date().getFullYear()}-01-01`).getTime() / 1000,
                  new Date(`${new Date().getFullYear() + 1}-01-01`).getTime() / 1000,
                ]
              : undefined),
        },
      },
      series,
      tzDate: (ts) => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC'),
    };
  }, [
    chartKey,
    dateRange,
    formatValues,
    groupBy,
    height,
    hoursByLabel,
    hoursTotal,
    label,
    singlePoint,
  ]);

  if (!hasData) {
    return <Message title="No data to plot." type={MessageType.Empty} />;
  }

  return (
    <div className={css.base}>
      <UPlotChart data={chartData} options={chartOptions} />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
