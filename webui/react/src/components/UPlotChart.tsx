import React, { useEffect, useRef, useState } from 'react';
import uPlot, { AlignedData } from 'uplot';

import useResize from 'hooks/useResize';

export interface Options extends Omit<uPlot.Options, 'width'> {
  width?: number;
}

interface Props {
  data?: AlignedData;
  options?: Options;
}

const UPlotChart: React.FC<Props> = ({ data, options }: Props) => {
  const [ chart, setChart ] = useState<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);

  // Chart setup.
  useEffect(() => {
    if (!chartDivRef.current || !options) return;

    if (!options.width) options.width = chartDivRef.current.offsetWidth;

    const plotChart = new uPlot(options as uPlot.Options, [ [] ], chartDivRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ chartDivRef, options ]);

  // Chart data.
  useEffect(() => {
    if (!chart || !data) return;
    chart.setData(data);
  }, [ chart, data ]);

  // Resize the chart when resize events happen.
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chart || !options?.height) return;
    chart.setSize({ height: options.height, width: resize.width });
  }, [ chart, options?.height, resize ]);

  return <div ref={chartDivRef} />;
};

export default UPlotChart;
