import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import uPlot, { Options, Series } from 'uplot';

import useResize from 'hooks/useResize';
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
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ chart, setChart ] = useState<uPlot>();

  useEffect(() => {
    if (!chartRef.current) return;

    let dateFormat = 'MM-DD';
    let timeSeries: Series = { label: 'Day', value: '{YYYY}-{MM}-{DD}' };
    if (groupBy === GroupBy.Month) {
      dateFormat = 'YYYY-MM';
      timeSeries = { label: 'Month', value: '{YYYY}-{MM}' };
    }
    const timeUnix: number[] = time.map(item => Date.parse(item) / 1000);

    const data = [ timeUnix ];
    const series: Series[] = [ timeSeries ];
    if (hoursTotal) {
      data.push(hoursTotal);
      series.push({
        label: 'total',
        show: false,
        stroke: glasbeyColor(series.length - 1),
        width: 2,
      });
    }
    Object.keys(hoursByLabel).forEach(label => {
      data.push(hoursByLabel[label]);
      series.push({
        label,
        stroke: glasbeyColor(series.length - 1),
        width: 2,
      });
    });

    const options = {
      axes: [
        {
          grid: { show: false },
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
      width: chartRef.current.offsetWidth,
    } as Options;

    const plotChart = new uPlot(options, data, chartRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [
    groupBy,
    height,
    hoursByLabel,
    hoursTotal,
    time,
  ]);

  // Resize the chart when resize events happen.
  useEffect(() => {
    if (chart) chart.setSize({ height, width: resize.width });
  }, [ chart, height, resize ]);

  return (
    <div className={css.base}>
      <div ref={chartRef} />
    </div>
  );
};

export default ClusterHistoricalUsageChart;
