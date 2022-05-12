import React, { MutableRefObject, useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import usePrevious from 'hooks/usePrevious';
import useResize from 'hooks/useResize';
import Message, { MessageType } from 'shared/components/message';
import handleError from 'utils/error';

import { ErrorLevel, ErrorType } from '../../shared/utils/error';

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
}

interface ChartBounds {
  isZoomed?: boolean;
  max?: number;
  min?: number;
}

type ChartBoundsData = Record<string, ChartBounds>

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

const getExtendedOptions = (
  options: Partial<uPlot.Options> | undefined,
  boundsRef: MutableRefObject<ChartBoundsData>,
  width?: number,
  onReady?: (uPlot: uPlot) => void,
  onDestroy?: (uPlot: uPlot) => void,
): uPlot.Options =>
  uPlot.assign(
    {
      hooks: {
        destroy: [ onDestroy ],
        ready: [ onReady ],
        setScale: [
          (uPlot: uPlot, scaleKey: string) => {
            /**
             * This fires *after* scale change
             * This function attempts to detect whether update was due to a zoom.
             * It does so by looking at whether the bounds curresponding to the
             * chart viewport are smaller than the maximum previous extent of those
             * bounds. If so, it is "zoomed". Sometimes setData can cause this to happen.
             * In particular, the scales generated for two points may be smaller than
             * for the first point alone. We try to provide some help here by saying
             * that if there is only one data point and an xScale is provided in options,
             * then don't reset the scale in setData. But the consumer of the component
             * still would need to manually provide a scale with min and max
             * centered around the point when- and only when- their component has one point
             */
            if (scaleKey !== 'x') return;

            // prevMax/Min indicate the previous max/min extent of the data.
            const prevMax = boundsRef.current[scaleKey]?.max;
            const prevMin = boundsRef.current[scaleKey]?.min;

            const bboxMax: number | undefined = uPlot.posToVal(
              scaleKey === 'x' ? uPlot.bbox.width : 0,
              scaleKey,
            );
            const bboxMin: number | undefined = uPlot.posToVal(
              scaleKey === 'x' ? 0 : uPlot.bbox.height,
              scaleKey,
            );

            // the new max/min extent of the data
            const maxMax = Math.max(prevMax ?? Number.MIN_SAFE_INTEGER, bboxMax);
            const minMin = Math.min(prevMin ?? Number.MAX_SAFE_INTEGER, bboxMin);

            /**
             * here we are cheating a bit by assuming that a zoom is smaller on both ends
             * this is to get around the issue of calling it a zoom when setData
             * causes the scale to go from [99.9, 100.1] to [100, 200]
             */
            const isZoomed = bboxMax < maxMax && bboxMin > minMin;

            boundsRef.current[scaleKey] = {
              isZoomed,
              max: maxMax,
              min: minMin,
            };
          },
        ],
      },
      width,
    },
    options || {},
  ) as uPlot.Options;

const UPlotChart: React.FC<Props> = ({
  data,
  focusIndex,
  options,
  style,
  noDataMessage,
}: Props) => {
  const chartRef = useRef<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const boundsRef = useRef<Record<string, ChartBounds>>({});
  const [ isReady, setIsReady ] = useState(false);

  const hasData = data && data.length > 1 && (options?.mode === 2 || data?.[0]?.length);

  const extendedOptions = useMemo(
    () =>
      getExtendedOptions(
        options,
        boundsRef,
        chartDivRef.current?.offsetWidth,
        () => setIsReady(true),
        () => setIsReady(false),
      ),
    [ options ],
  );
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

      const isZoomed = Object.values(boundsRef.current).some((i) => i.isZoomed === true);
      boundsRef.current = {};
      if (!isZoomed) {
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
      const isZoomed: boolean = Object.values(boundsRef.current).some((i) => i.isZoomed === true);
      const propsProvideScales: boolean = Object.values(extendedOptions?.scales ?? {}).some(
        (scale) => scale.min != null || scale.max != null,
      );
      const onlyOneXValue: boolean = Array.isArray(data) && data[0]?.length === 1;
      const customScaleIsInEffect = isZoomed || (propsProvideScales && onlyOneXValue);
      const resetScales = !customScaleIsInEffect;

      try {
        if (chartRef.current && isReady){
          chartRef.current.setData(data as AlignedData, resetScales);
          if (onlyOneXValue) chartRef.current.redraw(true, false);
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
  }, [ previousExtendedOptions, extendedOptions, data, isReady ]);

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
