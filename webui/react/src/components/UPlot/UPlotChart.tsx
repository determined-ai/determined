import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import usePrevious from 'hooks/usePrevious';
import useResize from 'hooks/useResize';
import Message, { MessageType } from 'shared/components/message';
import { isEqual } from 'utils/data';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

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

interface ZoomInfo {
  isZoomed?: boolean;
  max?: number;
  min?: number;
}

type ZoomScales = Record<string, ZoomInfo>;

const EMPTY_DATA = [ [], [ [] ] ] as unknown as uPlot.AlignedData;
const SCROLL_THROTTLE_TIME = 500;

const shouldRecreate = (
  chart?: uPlot,
  oldOptions?: Partial<uPlot.Options>,
  newOptions?: Partial<uPlot.Options>,
): boolean => {
  if (!chart || !oldOptions) return true;
  if (!newOptions) return false;
  if (oldOptions.id !== newOptions.id) return true;
  if (oldOptions.title !== newOptions.title) return true;
  if (oldOptions.axes?.length !== newOptions.axes?.length) return true;
  if (oldOptions.scales?.y?.distr !== newOptions.scales?.y?.distr) return true;

  const oldOptionKeys = JSON.stringify(Object.keys(oldOptions).sort());
  const newOptionKeys = JSON.stringify(Object.keys(newOptions).sort());
  if (oldOptionKeys !== newOptionKeys) return true;

  const someAxisLabelHasChanged = oldOptions.axes?.some((oldAxis, seriesIdx) => {
    return oldAxis.label !== newOptions.axes?.[seriesIdx]?.label;
  });
  if (someAxisLabelHasChanged) return true;

  return false;
};

const getSeriesMap = (series: uPlot.Series[] = []) => {
  return series.reduce((acc, current, index) => {
    if (current?.label) acc[current.label] = { index, series: current };
    return acc;
  }, {} as Record<string, { index: number, series: uPlot.Series }>);
};

const UPlotChart: React.FC<Props> = ({
  data,
  focusIndex,
  options,
  style,
  noDataMessage,
}: Props) => {
  const chartRef = useRef<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const zoomScalesRef = useRef<ZoomScales>({});
  const dataRef = useRef<uPlot.AlignedData>(EMPTY_DATA);
  const [ isEmpty, setIsEmpty ] = useState(true);
  const [ isReady, setIsReady ] = useState(false);
  const resize = useResize(chartDivRef);

  const fullOptions = useMemo(() => {
    return uPlot.assign({
      hooks: {
        destroy: [ () => setIsReady(false) ],
        init: [ (u: uPlot) => {
          u.over.ondblclick = () => {
            // Reset zoom for every scale key.
            Object.values(zoomScalesRef.current).forEach(zoomInfo => {
              zoomInfo.isZoomed = false;
            });
          };
        } ],
        ready: [ () => setIsReady(true) ],
        setScale: [ (u: uPlot, scaleKey: string) => {
          /**
           * Initial `setScale` simply initializes the zoom scales with the
           * min/max ranges for each `scaleKey`. Subsequent `setScale` will
           * be used to figure out if the view port has shrunk compared to
           * before. If so, it is considered a zoom.
           */
          const scale = u.scales[scaleKey];
          const isVertical = scale.ori === 1;
          const currentMax = zoomScalesRef.current[scaleKey]?.max;
          const currentMin = zoomScalesRef.current[scaleKey]?.min;
          // Top resolves to max vertical value, and top + height resolves to min vertical value.
          const maxPos = isVertical ? u.bbox.top : u.bbox.left + u.bbox.width;
          const minPos = isVertical ? u.bbox.top + u.bbox.height : u.bbox.left;
          const newMax = u.posToVal(maxPos, scaleKey);
          const newMin = u.posToVal(minPos, scaleKey);

          if (currentMax == null || currentMin == null) {
            zoomScalesRef.current[scaleKey] = { isZoomed: false, max: newMax, min: newMin };
          } else {
            zoomScalesRef.current[scaleKey] = {
              isZoomed: newMax < currentMax || newMin > currentMin,
              max: Math.min(currentMax, newMax),
              min: Math.max(currentMin, newMin),
            };
          }
        } ],
      },
      width: chartDivRef.current?.offsetWidth,
    }, options ?? {}) as uPlot.Options;
  }, [ options ]);

  const prevFullOptions = usePrevious(fullOptions, undefined);

  /**
   * Ensure that the chart is cleaned up during unmount if applicable.
   */
  useEffect(() => {
    return () => {
      chartRef?.current?.destroy();
      chartRef.current = undefined;
    };
  }, []);

  /**
   * Recreate chart if the new `options` prop changes require it.
   */
  useEffect(() => {
    if (chartDivRef.current && shouldRecreate(chartRef.current, prevFullOptions, fullOptions)) {
      chartRef?.current?.destroy();
      chartRef.current = undefined;

      try {
        chartRef.current = new uPlot(fullOptions, dataRef.current, chartDivRef.current);
      } catch (e) {
        // Something happened during uPlot creation, clear out any uPlot artifacts.
        chartRef?.current?.destroy();
        chartRef.current = undefined;

        // Set as "no data" for now.
        setIsEmpty(true);

        // Record the error.
        handleError(e, {
          level: ErrorLevel.Warn,
          publicSubject: 'Most likely bad data structure.',
          type: ErrorType.Input,
        });
      }
    }
  }, [ fullOptions, prevFullOptions ]);

  /**
   * Update `isEmpty` state and `dataRef` when source data changes.
   */
  useEffect(() => {
    const alignedData = data as uPlot.AlignedData | undefined;

    if (fullOptions.mode === 2 && alignedData) {
      setIsEmpty(false);
      dataRef.current = alignedData;
    } else if (alignedData && alignedData?.length >= 2) {
      const xDataCount = alignedData[0]?.length ?? 0;
      setIsEmpty(xDataCount === 0);
      dataRef.current = alignedData;
    } else {
      setIsEmpty(false);
      dataRef.current = EMPTY_DATA;
    }

    // Update chart with data changes if chart already exists.
    if (chartRef.current) {
      const isZoomed = Object.values(zoomScalesRef.current).some(i => i.isZoomed);
      chartRef.current?.setData(dataRef.current, !isZoomed);
    }
  }, [ data, fullOptions.mode, isReady ]);

  /**
   * When scales settings change in options, update chart scale.
   */
  useEffect(() => {
    const oldScaleKeys = Object.keys(chartRef.current?.scales ?? {}).sort();
    const newScales = fullOptions.scales ?? {};
    const newScaleKeys = Object.keys(newScales).sort();
    if (isEqual(oldScaleKeys, newScaleKeys)) return;

    newScaleKeys.forEach(scaleKey => {
      const scales = newScales[scaleKey];
      if (scales.min != null && scales.max != null) {
        chartRef.current?.setScale(scaleKey, { max: scales.max, min: scales.min });
      }
    });
  }, [ fullOptions.scales ]);

  /**
   * When series changes add or delete series.
   */
  useEffect(() => {
    if (fullOptions.mode === 2) return;

    const oldSeriesMap = getSeriesMap(chartRef.current?.series);
    const newSeriesMap = getSeriesMap(fullOptions.series);

    // Add new series that currently don't exist in the chart.
    Object.keys(newSeriesMap).forEach(label => {
      if (oldSeriesMap[label]) return;
      chartRef.current?.addSeries(newSeriesMap[label].series);
    });

    // Remove existing series that no longer exists in `options.series`.
    Object.keys(oldSeriesMap).forEach(label => {
      if (newSeriesMap[label]) return;
      chartRef.current?.delSeries(oldSeriesMap[label].index);
    });
  }, [ fullOptions.mode, fullOptions.series ]);

  /**
   * Resize the chart when resize events happen.
   */
  useEffect(() => {
    if (!chartRef.current || !resize.width || !resize.height) return;
    const [ width, height ] = [ resize.width, fullOptions.height || chartRef.current.height ];

    if (chartRef.current.width === width && chartRef.current.height === height) return;

    chartRef.current.setSize({ height, width });
  }, [ fullOptions.height, resize ]);

  /**
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    const hasFocus = focusIndex !== undefined;
    chartRef.current?.setSeries(hasFocus ? focusIndex as number + 1 : null, { focus: hasFocus });
  }, [ focusIndex ]);

  /**
   * Resync the chart when scroll events happen to correct the cursor position upon
   * a parent container scrolling.
   */
  useEffect(() => {
    const throttleFunc = throttle(SCROLL_THROTTLE_TIME, () => {
      chartRef.current?.syncRect();
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
      {isEmpty && (
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
