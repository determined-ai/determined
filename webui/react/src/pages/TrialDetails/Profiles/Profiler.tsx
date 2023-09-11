import dayjs from 'dayjs';
import React from 'react';
import uPlot from 'uplot';

import { glasbeyColor } from 'components/kit/utils/color';
import { SyncProvider } from 'components/UPlot/SyncProvider';
import { TrialDetails } from 'types';

import SystemMetricChart from './Charts/SystemMetricChart';
import ThroughputMetricChart from './Charts/ThroughputMetricChart';
import TimingMetricChart from './Charts/TimingMetricChart';
import css from './Profiler.module.scss';

export interface Props {
  trial: TrialDetails;
}

export const CHART_HEIGHT = 300;

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
      <ThroughputMetricChart trial={trial} />
      <TimingMetricChart trial={trial} />
      <SystemMetricChart trial={trial} />
    </SyncProvider>
  );
};

export default Profiler;
