import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import uPlot, { AlignedData } from 'uplot';

import Spinner from 'components/Spinner';
import UPlotChart, { Options } from 'components/UPlotChart';
import { CHART_HEIGHT } from 'pages/TrialDetails/TrialDetailsProfiles';
import { TrialDetails } from 'types';
import { glasbeyColor } from 'utils/color';

import { FiltersInterface } from './SystemMetricFilter';
import {
  convertMetricsToUplotData, getUnitForMetricName, MetricType, useFetchMetrics,
} from './utils';

export interface Props {
  filters: FiltersInterface,
  trial: TrialDetails;
}

const SystemMetricChart: React.FC<Props> = ({ filters, trial }: Props) => {
  const [ chartData, setChartData ] = useState<AlignedData>();
  const [ chartOptions, setChartOptions ] = useState<Options>();
  const systemMetrics = useFetchMetrics(
    trial.id,
    MetricType.System,
    filters.name,
    filters.agentId,
    filters.gpuUuid,
  );

  const xMin = chartData && chartData[0] && chartData[0][0] ? chartData[0][0] : 0;

  useEffect(() => {
    setChartOptions({
      axes: [
        {
          space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
            const rangeSecs = scaleMax - scaleMin;
            const pxPerSec = plotDim / rangeSecs;
            return Math.max(60, pxPerSec * 10);
          },
          values: (self, splits) => {
            return splits.map(i => dayjs.utc(i * 1000).format('HH:mm:ss'));
          },
        },
        ...systemMetrics.names.map((name) => ({ label: getUnitForMetricName(name) })),
      ],
      height: CHART_HEIGHT,
      scales: xMin ? { x: { auto: false, max: xMin + (5 * 60), min: xMin } } : {},
      series: [
        { label: 'Time', value: '{HH}:{mm}:{ss}' },
        ...systemMetrics.names.map((name, index) => ({
          label: name,
          points: { show: false },
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
      tzDate: ts => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC'),
    });
  }, [ systemMetrics.names, xMin ]);

  useEffect(() => {
    setChartData(convertMetricsToUplotData(systemMetrics.dataByUnixTime));
  }, [ systemMetrics.dataByUnixTime ]);

  return (
    <Spinner spinning={systemMetrics.isLoading}>
      <UPlotChart data={chartData} options={chartOptions} />
    </Spinner>
  );
};

export default SystemMetricChart;
