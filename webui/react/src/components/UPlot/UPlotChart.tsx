// @ts-nocheck
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import usePrevious from 'hooks/usePrevious';
import useResize from 'hooks/useResize';
import Message, { MessageType } from 'shared/components/message';
import { isNumber } from 'utils/data';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import { useSyncableBounds } from './SyncableBounds';
import { FacetedData } from './types';

export interface Options extends Omit<uPlot.Options, 'width'> {
  key?: number;
  width?: number;
}

interface Props {
  data?: AlignedData | FacetedData;
  focusIndex?: number;
  noDataMessage?: string;
  options?: Partial<Options>;
  style?: React.CSSProperties;
  title?: string;
}

const SCROLL_THROTTLE_TIME = 500;

const shouldRecreate = (
  prev: Partial<Options> | undefined,
  next: Partial<Options> | undefined,
  chart: uPlot | undefined,
): boolean => {
  if (!chart) return true;
  if (!next) return false;
  if (!prev) return true;
  if (prev === next) return false;
  if (prev.key !== next.key) return true;
  if (Object.keys(prev).length !== Object.keys(next).length) return true;

  if (prev.title !== next.title) return true;
  if (prev.axes?.length !== next.axes?.length) return true;

  if (chart?.series?.length !== next.series?.length) return true;

  if (prev.scales?.y?.distr !== next.scales?.y?.distr) return true;

  const someAxisLabelHasChanged =
    prev.axes?.some((prevAxis, seriesIdx) => {
      const nextAxis = next.axes?.[seriesIdx];
      return prevAxis.label !== nextAxis?.label;
    });
  if (someAxisLabelHasChanged) return true;

  const someSeriesHasChanged =
    chart.series.some((chartSerie, seriesIdx) => {
      const nextSerie = next.series?.[seriesIdx];
      const prevSerie = prev.series?.[seriesIdx];
      return (
        (nextSerie?.show != null && chartSerie?.show !== nextSerie?.show)
        || (prevSerie?.show != null && prevSerie?.show !== nextSerie?.show)
        || (nextSerie?.label != null && chartSerie?.label !== nextSerie?.label)
        || (nextSerie?.fill != null && chartSerie?.fill !== nextSerie?.fill)
      );
    });
  if(someSeriesHasChanged) return true;

  return false;
};

const UPlotChart: React.FC<Props> = ({
  data,
  focusIndex,
  options,
  style,
  noDataMessage,
  title,
}: Props) => {
  const chartRef = useRef<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const [ isReady, setIsReady ] = useState(false);

  const { zoomed, boundsOptions } = useSyncableBounds();

  if (title === 'System Metrics') {
    window.sys = chartRef.current;
  } else if (title === 'Throughput') {
    window.thru = chartRef.current;
  } else if (title === 'Timing Metrics') {
    window.time = chartRef.current;
  }
  window.options = options;
  window.data = data;

  const hasData = data && data.length > 1 && (options?.mode === 2 || data?.[0]?.length);

  const extendedOptions = useMemo(() =>
    uPlot.assign(
      {
        hooks: {
          destroy: [ () => setIsReady(false) ],
          ready: [ () => setIsReady(true) ],

        },
        width: chartDivRef.current?.offsetWidth,
      },
      boundsOptions || {},
      options || {},
    ), [ options, boundsOptions ]);

  // console.log(extendedOptions);
  const previousExtendedOptions = usePrevious(extendedOptions, undefined);

  useEffect(() => {
    return () => {
      chartRef?.current?.destroy();
      chartRef.current = undefined;
    };
  }, []);

  useEffect(() => {
    if (!chartDivRef.current) return;
    if (shouldRecreate(previousExtendedOptions, extendedOptions, chartRef.current)) {
      /**
       * TODO: instead of returning true or false,
       * return a list of actions/payloads to dispatch
       * with setData, setSeries, addSeries, etc.
       */
      chartRef.current?.destroy();
      chartRef.current = undefined;

      if (!zoomed) {
        /**
         * reset of zoom
         */
      } else {
        /**
         * TODO: preserve zoom when new series is selected?
         * There are some additional challenges because the setDatas will be interpreted as
         * zooms when the data is streaming in since the bounds are smaller at first
         * Might also want to preserve other user interactions with the charts
         * by taking some things from chartRef.current and putting them in newOptions
         * e.g. a series is updated, say it's hidden, that update is reflected in options
         * but series is a list, and uPlot.assign does not merge the lists, it clobbers.
         * (as one would expect)
         */
      }
      try {
        if (extendedOptions?.mode === 2 || extendedOptions.series.length === data?.length){
          chartRef.current = new uPlot(extendedOptions, data as AlignedData, chartDivRef.current);
          chartRef.current.title = title;
        }
      } catch(e) {
        chartRef.current?.destroy();
        chartRef.current = undefined;
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Unable to Load data for chart',
          publicSubject: 'Bad Data',
          silent: false,
          type: ErrorType.Ui,
        });
      }
    } else {
      try {
        if (chartRef.current && isReady){
          // chopping the data for throughput to test the sync
          const disp =
          title === 'Throughput'
            ? data.map((ser) => ser.slice(0, Math.floor(ser.length / 2)))
            : data;
          const min = new Date(Math.min(...disp[0]?.filter(isNumber)));
          const max = new Date(Math.max(...disp[0]?.filter(isNumber)));
          chartRef.current.setData(disp as AlignedData, false);
        }

      } catch(e) {
        chartRef.current?.destroy();
        chartRef.current = undefined;
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Unable to Load data for chart',
          publicSubject: 'Bad Data',
          silent: false,
          type: ErrorType.Ui,
        });
      }
    }
  }, [ previousExtendedOptions, extendedOptions, data, isReady, title, zoomed ]);

  /**
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    if (!chartRef.current) return;
    const hasFocus = focusIndex !== undefined;
    chartRef.current.setSeries(hasFocus ? (focusIndex as number) + 1 : null, { focus: hasFocus });
  }, [ focusIndex ]);

  /*
   * Resize the chart when resize events happen.
   */
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chartRef.current) return;
    const [ width, height ] = [ resize.width, options?.height || chartRef.current.height ];
    if (chartRef.current.width === width && chartRef.current.height === height) return;
    chartRef.current.setSize({ height, width });
  }, [ options?.height, resize ]);

  /*
   * Resync the chart when scroll events happen to correct the cursor position upon
   * a parent container scrolling.
   */
  useEffect(() => {
    const throttleFunc = throttle(SCROLL_THROTTLE_TIME, () => {
      if (chartRef.current) chartRef.current.syncRect();
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
  }, []);

  return (
    <div ref={chartDivRef} style={style}>
      {!hasData && (
        <Message
          style={{ height: options?.height ?? 'auto' }}
          title={noDataMessage || 'No Data to plot.'}
          type={MessageType.Empty}
        />
      )}
    </div>
  );
};

export default UPlotChart;
