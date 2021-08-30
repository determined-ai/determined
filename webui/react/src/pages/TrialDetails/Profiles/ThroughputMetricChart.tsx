import dayjs from 'dayjs';
import React, { useMemo } from 'react';
import uPlot, { AlignedData } from 'uplot';

import UPlotChart, { Options } from 'components/UPlotChart';
import { CHART_HEIGHT } from 'pages/TrialDetails/TrialDetailsProfiles';
import { glasbeyColor } from 'utils/color';

import {
  convertMetricsToUplotData, getUnitForMetricName, MetricsAggregateInterface,
} from './utils';

export interface Props {
  throughputMetrics: MetricsAggregateInterface;
}

const ThroughputMetricChart: React.FC<Props> = ({ throughputMetrics }: Props) => {
  const chartData: AlignedData = useMemo(() => {
    return convertMetricsToUplotData(throughputMetrics.dataByUnixTime, throughputMetrics.names);
  }, [ throughputMetrics.dataByUnixTime, throughputMetrics.names ]);

  const xMin = useMemo(() => {
    return chartData && chartData[0] && chartData[0][0] ? chartData[0][0] : 0;
  }, [ chartData ]);

  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        {
          label: 'Time',
          space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
            const rangeMs = scaleMax - scaleMin;
            const msPerSec = plotDim / rangeMs;
            return Math.max(60, msPerSec * 10 * 1000);
          },
          values: (self, splits) => {
            return splits.map(i => dayjs.utc(i).format('HH:mm:ss'));
          },
        },
        {
          label: getUnitForMetricName('samples_per_second'),
          size: (self: uPlot, values: string[]) => {
            if (!values) return 50;
            const maxChars = Math.max(...values.map(el => el.toString().length));
            return 25 + Math.max(25, maxChars * 8);
          },
        },
      ],
      height: CHART_HEIGHT,
      scales: xMin
        ? { x: { auto: false, max: xMin + (5 * 60 * 1000), min: xMin, time: false } }
        : { x: { time: false } },
      series: [
        {
          label: 'Time',
          value: (self, rawValue) => {
            return dayjs.utc(rawValue).format('HH:mm:ss.SSS');
          },
        },
        ...throughputMetrics.names.map((name, index) => ({
          label: name,
          points: { show: false },
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
      tzDate: ts => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC'),
    };
  }, [ throughputMetrics.names, xMin ]);

  return <UPlotChart data={chartData} options={chartOptions} />;
};

export default ThroughputMetricChart;
