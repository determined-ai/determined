import React, { useEffect, useRef, useState } from 'react';
import uPlot, { Options } from 'uplot';

import Spinner from 'components/Spinner';
import useResize from 'hooks/useResize';
import { CHART_HEIGHT } from 'pages/TrialDetails/TrialDetailsProfiles';
import { TrialDetails } from 'types';
import { glasbeyColor } from 'utils/color';
import { findFactorOfNumber } from 'utils/number';

import { convertMetricsToUplotData, MetricType, useFetchMetrics } from './utils';

export interface Props {
  trial: TrialDetails;
}

const TimingMetricChart: React.FC<Props> = ({ trial }: Props) => {
  const [ chart, setChart ] = useState<uPlot>();
  const chartRef = useRef<HTMLDivElement>(null);
  const timingMetrics = useFetchMetrics(trial.id, MetricType.Timing);

  useEffect(() => {
    if (!chartRef.current) return;

    const options = {
      axes: [
        {
          space: (self, axisIdx, scaleMin, scaleMax, plotDim) => {
            const range = scaleMax - scaleMin + 1;
            const factor = findFactorOfNumber(range).reverse()
              .find(factor => plotDim / factor > 35);
            return factor ? (plotDim / factor) : 35;
          },
        },
        { label: 'Milliseconds' },
      ],
      height: CHART_HEIGHT,
      scales: { x: { time: false } },
      series: [
        { label: 'Batch' },
        ...timingMetrics.names.map((name, index) => ({
          label: name,
          points: { show: false },
          stroke: glasbeyColor(index),
          width: 2,
        })),
      ],
      tzDate: ts => uPlot.tzDate(new Date(ts * 1e3), 'Etc/UTC'),
      width: chartRef.current.offsetWidth,
    } as Options;

    const plotChart = new uPlot(options, [], chartRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ chartRef, timingMetrics.names ]);

  useEffect(() => {
    if (!chart) return;
    chart.setData(convertMetricsToUplotData(timingMetrics.dataByBatch));
  }, [ chart, timingMetrics ]);

  // Resize the chart when resize events happen.
  const resize = useResize(chartRef);
  useEffect(() => {
    if (chart) chart.setSize({ height: CHART_HEIGHT, width: resize.width });
  }, [ chart, resize ]);

  return (
    <Spinner spinning={timingMetrics.isLoading}>
      <div ref={chartRef} />
    </Spinner>
  );
};

export default TimingMetricChart;
