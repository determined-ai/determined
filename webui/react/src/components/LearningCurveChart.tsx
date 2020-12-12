import React, { useCallback, useEffect, useRef } from 'react';
import uPlot, { AlignedData, Options } from 'uplot';

import { generateTrials } from 'utils/chart';
import { numericSorter } from 'utils/data';

import 'uplot/dist/uPlot.min.css';

interface DataPoint {
  color: number;
  x: number;
  y: number;
}

interface Props {
  data: (number | null)[][];
  trialIds: number[];
  xValues: number[];
}

const UPLOT_OPTIONS = {
  axes: [
    {
      grid: { width: 1 },
      scale: 'x',
      side: 2,
    },
    {
      grid: { width: 1 },
      scale: 'metric',
      side: 3,
    },
  ],
  cursor: { focus: { prox: 30 } },
  focus: { alpha: 0.3 },
  height: 400,
  legend: { show: false },
  scales: {
    metric: { auto: true, time: false },
    x: { time: false },
  },
  series: [ { label: 'batches' } ],
};

const LearningCurveChart: React.FC<Props> = ({ data, trialIds, xValues }: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  console.log('data', data);
  console.log('xValues', xValues);

  const handleCursorChange = useCallback((
    plot: uPlot,
    seriesIdx: number,
    closestIdx: number,
    xValue: number,
  ) => {
    // console.log('cursor changed', seriesIdx, closestIdx, xValue);
    return closestIdx;
  }, []);

  useEffect(() => {
    if (!chartRef.current) return;

    const now = Date.now();
    const options = uPlot.assign({}, UPLOT_OPTIONS, {
      cursor: { dataIdx: handleCursorChange },
      series: [
        { label: 'batches' },
        ...trialIds.map(trialId => ({
          label: `trial ${trialId}`,
          scale: 'metric',
          spanGaps: true,
          stroke: 'rgba(50, 0, 150, 1.0)',
          width: 1 / devicePixelRatio,
        })),
      ],
      width: chartRef.current.offsetWidth,
    }) as Options;

    const chart = new uPlot(options, [ xValues, ...data ], chartRef.current);
    console.log('render time', (Date.now() - now) / 1000);

    return () => chart.destroy();
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ data, xValues ]);

  return <div ref={chartRef} />;
};

export default LearningCurveChart;
