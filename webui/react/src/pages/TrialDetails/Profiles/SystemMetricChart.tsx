import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import uPlot, { Options } from 'uplot';

import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
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
  const [ chart, setChart ] = useState<uPlot>();
  const chartRef = useRef<HTMLDivElement>(null);
  const systemMetrics = useFetchMetrics(
    trial.id,
    MetricType.System,
    filters.name,
    filters.agentId,
    filters.gpuUuid,
  );

  useEffect(() => {
    if (!chartRef.current) return;

    const options = {
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
      width: chartRef.current.offsetWidth,
    } as Options;

    const plotChart = new uPlot(options, [ [] ], chartRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ chartRef, systemMetrics.names ]);

  useEffect(() => {
    if (!chart) return;
    const data = convertMetricsToUplotData(systemMetrics.dataByUnixTime);

    const xMin = data[0][0] || 0;

    chart.setScale('x', { max: xMin + (5 * 60), min: xMin });
    chart.setData(data, false);
  }, [ chart, systemMetrics ]);

  // Resize the chart when resize events happen.
  const resize = useResize(chartRef);
  useEffect(() => {
    if (chart) chart.setSize({ height: CHART_HEIGHT, width: resize.width });
  }, [ chart, resize ]);

  return (
    <Spinner spinning={systemMetrics.isLoading}>
      <div ref={chartRef} />
    </Spinner>
  );
};

export default SystemMetricChart;
