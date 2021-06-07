import React, {
  forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState,
} from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import Message, { MessageType } from 'components/Message';
import useResize from 'hooks/useResize';

export interface Options extends Omit<uPlot.Options, 'width'> {
  width?: number;
}

interface SerieMaxMin {
  max?: number;
  min?: number;
}

interface Props {
  data?: AlignedData;
  options?: Options;
  ref?: React.Ref<uPlot|undefined>;
}

const SCROLL_THROTTLE_TIME = 500;

const UPlotChart: React.FC<Props> = forwardRef((
  { data, options }: Props,
  ref?: React.Ref<uPlot|undefined>,
) => {
  const [ chart, setChart ] = useState<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const isZoomed = useRef<boolean>(false);

  const hasData: boolean = useMemo(() => {
    // no x values
    if (!data || !data[0] || data[0].length === 0) return false;

    // series values length not matching x values length
    for (let i = 1; i < data.length; i++) {
      if (data[i].length !== data[0].length) return false;
    }

    return true;
  }, [ data ]);

  /*
   * Chart setup.
   */
  useEffect(() => {
    if (!chartDivRef.current || !hasData || !options) return;

    const seriesMaxMin: Record<string, SerieMaxMin> = {};

    const optionsExtended = uPlot.assign(
      {
        cursor: { drag: { dist: 5, uni: 10, x: true, y: true } },
        hooks: {
          ready: [ (chart: uPlot) => setChart(chart) ],
          setScale: [ (uPlot: uPlot, scaleKey: string) => {
            const scaleSeries = uPlot.series.filter(serie => serie.scale === scaleKey);

            let currentMax = undefined;
            let currentMin = undefined;
            let max: number|undefined = seriesMaxMin[scaleKey]?.max;
            let min: number|undefined = seriesMaxMin[scaleKey]?.min;

            scaleSeries.forEach(serie => {
              currentMax = serie.max;
              currentMin = serie.min;
              if (serie.max != null && (max == null || serie.max > max)) max = serie.max;
              if (serie.min != null && (min == null || serie.min < min)) min = serie.min;
            });

            seriesMaxMin[scaleKey] = { max, min };
            if (currentMax != null
              && max != null
              && currentMin != null
              && min != null) {
              isZoomed.current = (currentMax < max || currentMin > min);
            }
          } ],
        },
        width: chartDivRef.current.offsetWidth,
      },
      options,
    );

    const plotChart = new uPlot(optionsExtended as uPlot.Options, [ [] ], chartDivRef.current);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ chartDivRef, hasData, options ]);

  /*
   * Chart data.
   */
  useEffect(() => {
    if (!chart || !data) return;
    chart.setData(data, !isZoomed.current);
  }, [ chart, data ]);

  /*
   * Resize the chart when resize events happen.
   */
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chart || !options?.height || !resize.width) return;
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

  useImperativeHandle(ref, () => chart, [ chart ]);

  return hasData
    ? <div ref={chartDivRef} />
    : <Message title="No data to plot." type={MessageType.Empty} />;
});

export default UPlotChart;
