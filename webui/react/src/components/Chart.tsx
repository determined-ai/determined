import React, { useEffect, useRef, useState } from 'react';
import uPlot, { Options } from 'uplot';

import useResize from 'hooks/useResize';
import { generateScatter } from 'utils/chart';
import { numericSorter } from 'utils/data';

import css from './Chart.module.scss';

interface Props {
  height?: number;
}

const CHART_HEIGHT = 200;
const UPLOT_OPTIONS = {
  axes: [
    {
      grid: { width: 1 },
      scale: 'x',
      side: 2,
    },
    {
      grid: { width: 1 },
      scale: 'y',
      side: 3,
    },
  ],
  legend: { show: false },
  scales: {
    x: { auto: true, time: false },
    y: { auto: true, time: false },
  },
  series: [ { label: 'x' } ],
};

const Chart: React.FC<Props> = ({
  height = CHART_HEIGHT,
  ...props
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ chart, setChart ] = useState<uPlot>();

  useEffect(() => {
    if (!chartRef.current) return;

    const data: number[][] = [];
    const options = uPlot.assign({}, UPLOT_OPTIONS, {
      height,
      width: chartRef.current.offsetWidth,
    }) as Options;

    const [ xData, yData ] = generateScatter(100).reduce((acc, point) => {
      acc[0].push(point.x);
      acc[1].push(point.y);
      return acc;
    }, [ [] as number[], [] as number[] ]);

    data.push(xData.sort(numericSorter), yData.sort(numericSorter));
    options.series.push({
      label: 'scatter',
      scale: 'y',
      stroke: 'rgba(50, 0, 255, 1.0)',
      width: 1 / devicePixelRatio,
    });

    const plotChart = new uPlot(options, [ xData, yData ], chartRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ height ]);

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

export default Chart;
