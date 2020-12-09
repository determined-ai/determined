import React, { useCallback, useEffect, useRef } from 'react';
import uPlot, { Options } from 'uplot';

import { generateTrials } from 'utils/chart';
import { numericSorter } from 'utils/data';

import 'uplot/dist/uPlot.min.css';

interface DataPoint {
  color: number;
  x: number;
  y: number;
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

const LearningCurveChart: React.FC = () => {
  const chartRef = useRef<HTMLDivElement>(null);

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
    console.log('render start...');

    console.log('size', chartRef.current.offsetWidth);
    const now = Date.now();
    const data: number[][] = [];
    const options = uPlot.assign(UPLOT_OPTIONS, {
      cursor: { dataIdx: handleCursorChange },
      width: chartRef.current.offsetWidth,
    }) as Options;

    const xMap: Record<string, boolean> = {};
    generateTrials().forEach((trial, index) => {
      const series: number[] = [];
      trial.forEach(point => {
        const xKey = point.x.toString();
        if (xMap[xKey] == null) xMap[xKey] = true;
        series.push(point.y);
      });
      data.push(series);
      options.series.push({
        label: `trial ${index}`,
        scale: 'metric',
        stroke: 'rgba(50, 0, 255, 1.0)',
        width: 1 / devicePixelRatio,
      });
    });

    const xValues = Object.keys(xMap).map(x => parseFloat(x)).sort(numericSorter);

    const chart = new uPlot(options, [ xValues, ...data ], chartRef.current);

    console.log('xMap', xMap);
    console.log('generateTrials', xValues);
    console.log('render ended', (Date.now() - now) / 1000);

    return () => {
      if (chart) chart.destroy();
    };
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, []);

  return <div ref={chartRef} />;
};

export default LearningCurveChart;
