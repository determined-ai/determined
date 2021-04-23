import React, { useEffect, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import useResize from 'hooks/useResize';

export interface Options extends Omit<uPlot.Options, 'width'> {
  width?: number;
}

interface Props {
  data?: AlignedData;
  options?: Options;
}

const SCROLL_THROTTLE_TIME = 500;

const UPlotChart: React.FC<Props> = ({ data, options }: Props) => {
  const [ chart, setChart ] = useState<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);

  /*
   * Chart setup.
   */
  useEffect(() => {
    if (!chartDivRef.current || !options) return;

    const optionsExtended = uPlot.assign(
      {
        cursor: { drag: { dist: 10, uni: 100, x: true, y: true } },
        width: chartDivRef.current.offsetWidth,
      },
      options,
    );

    const plotChart = new uPlot(optionsExtended as uPlot.Options, [ [] ], chartDivRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ chartDivRef, options ]);

  /*
   * Chart data.
   */
  useEffect(() => {
    if (!chart || !data) return;
    chart.setData(data);
  }, [ chart, data ]);

  /*
   * Resize the chart when resize events happen.
   */
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chart || !options?.height) return;
    chart.setSize({ height: options.height, width: resize.width });
  }, [ chart, options?.height, resize ]);

  /*
   * Resync the chart when scroll events happen to correct the cursor position upon
   * a parent container scrolling.
   */
  useEffect(() => {
    const throttleFunc = throttle(SCROLL_THROTTLE_TIME, () => {
      if (chart) chart.syncRect();
    });
    const handleScroll = () => throttleFunc();

    /*
     * The true at the end is the important part,
     * it tells the browser to capture the event on dispatch,
     * even if that event does not normally bubble, like change, focus, and scroll.
     */
    document.addEventListener('scroll', handleScroll, true);

    return () => {
      document.removeEventListener('scroll', handleScroll);
      throttleFunc.cancel();
    };
  }, [ chart ]);

  return <div ref={chartDivRef} />;
};

export default UPlotChart;
