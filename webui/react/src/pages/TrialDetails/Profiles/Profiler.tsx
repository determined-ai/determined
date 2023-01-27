import dayjs from 'dayjs';
import React from 'react';
import uPlot from 'uplot';

import { SyncProvider } from 'components/UPlot/SyncProvider';
import { Options } from 'components/UPlot/UPlotChart';
import { glasbeyColor } from 'shared/utils/color';
import { TrialDetails } from 'types';

import SystemMetricChart from './Charts/SystemMetricChart';
import ThroughputMetricChart from './Charts/ThroughputMetricChart';
import TimingMetricChart from './Charts/TimingMetricChart';
import css from './Profiler.module.scss';
import { getUnitForMetricName } from './utils';

export const CHART_HEIGHT = 300;

export interface Props {
  trial: TrialDetails;
}

/*
 * Shared uPlot chart options.
 */

const getOptionsForMetrics = (metricName: string, seriesNames: string[]): Partial<Options> => {
  return {
    axes: [timeAxis, getAxisForMetricName(metricName)],
    height: CHART_HEIGHT,
    scales: { x: { time: false } },
    series: [
      baseSeries.time,
      baseSeries.batch,
      ...seriesNames.slice(1).map(getSeriesForSeriesName), // 0th is batch
    ],
    tzDate,
  };
};

export const tzDate = (ts: number): Date => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC');

export const timeAxis: uPlot.Axis = {
  label: 'Time',
  scale: 'x',
  space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
    const rangeMs = scaleMax - scaleMin;
    const msPerSec = plotDim / rangeMs;
    return Math.max(60, msPerSec * 10 * 1000);
  },
  values: (self, splits) => {
    return splits.map((i) => dayjs.utc(i).format('HH:mm:ss'));
  },
};

const getAxisForMetricName = (metricName = '') => ({
  label: getUnitForMetricName(metricName),
  scale: 'y',
  size: (self: uPlot, values: string[]) => {
    if (!values) return 50;
    const maxChars = Math.max(...values.map((el) => el.toString().length));
    return 25 + Math.max(25, maxChars * 8);
  },
});

export const baseSeries: Record<string, uPlot.Series> = {
  batch: {
    class: css.disabledLegend,
    label: 'Batch',
    scale: 'y',
    show: false,
  },
  time: {
    label: 'Time',
    scale: 'x',
    value: (self, rawValue) => dayjs.utc(rawValue).format('HH:mm:ss.SSS').slice(0, -2),
  },
};

export const getSeriesForSeriesName = (name: string, index: number): uPlot.Series => ({
  label: name,
  points: { show: false },
  scale: 'y',
  spanGaps: true,
  stroke: glasbeyColor(index),
  width: 2,
});

const Profiler: React.FC<Props> = ({ trial }) => {
  return (
    <SyncProvider>
      <ThroughputMetricChart getOptionsForMetrics={getOptionsForMetrics} trial={trial} />
      <TimingMetricChart getOptionsForMetrics={getOptionsForMetrics} trial={trial} />
      <SystemMetricChart getOptionsForMetrics={getOptionsForMetrics} trial={trial} />
    </SyncProvider>
  );
};

export default Profiler;
