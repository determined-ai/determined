import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import usePrevious from 'hooks/usePrevious';
import useResize from 'hooks/useResize';
import Message, { MessageType } from 'shared/components/message';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

import { FacetedData } from './types';

export interface Options extends Omit<uPlot.Options, 'width'> {
  width?: number;
}

interface Props {
  data?: AlignedData | FacetedData;
  focusIndex?: number;
  noDataMessage?: string;
  options?: Partial<Options>;
  style?: React.CSSProperties;
}

interface Scale {
  isZoomed?: boolean;
  max?: number;
  min?: number;
}

const EMPTY_DATA = [ [], [ [] ] ] as unknown as uPlot.AlignedData;
const SCROLL_THROTTLE_TIME = 500;
const UPLOT_ERROR = {
  level: ErrorLevel.Warn,
  publicSubject: 'Something went wrong with uPlot.',
  type: ErrorType.Input,
};

const shouldRecreate = (
  newOptions: Partial<uPlot.Options>,
  oldOptions?: Partial<uPlot.Options>,
): boolean => {
  if (!oldOptions) return true;
  if (oldOptions.id !== newOptions.id) return true;
  if (oldOptions.title !== newOptions.title) return true;
  if (oldOptions.axes?.length !== newOptions.axes?.length) return true;
  if (oldOptions.plugins?.length !== newOptions.plugins?.length) return true;
  if (oldOptions.scales?.y?.distr !== newOptions.scales?.y?.distr) return true;

  /**
   * `uPlot.assign` changes the references of the plugins, so a simple
   * comparison of each plugin reference does not work. Instead, compare
   * the plugins converted into strings.
   */
  const oldPLugins = JSON.stringify(oldOptions.plugins);
  const newPLugins = JSON.stringify(newOptions.plugins);
  if (oldPLugins !== newPLugins) return true;

  const oldOptionKeys = JSON.stringify(Object.keys(oldOptions).sort());
  const newOptionKeys = JSON.stringify(Object.keys(newOptions).sort());
  if (oldOptionKeys !== newOptionKeys) return true;

  const someAxisLabelHasChanged = oldOptions.axes?.some((oldAxis, seriesIdx) => {
    return oldAxis.label !== newOptions.axes?.[seriesIdx]?.label;
  });
  if (someAxisLabelHasChanged) return true;

  return false;
};

const shouldRedraw = (
  newOptions: Partial<uPlot.Options>,
  oldOptions?: Partial<uPlot.Options>,
): boolean => {
  if (!oldOptions) return true;

  const someAxisStyleHasChanged = oldOptions.axes?.some((oldAxis, seriesIdx) => {
    const newAxis = newOptions.axes?.[seriesIdx] ?? {};
    const strokeChanged = oldAxis.stroke !== newAxis.stroke;
    const borderStrokeChanged = oldAxis.border?.stroke !== newAxis.border?.stroke;
    const gridStrokeChanged = oldAxis.grid?.stroke !== newAxis.grid?.stroke;
    const ticksStrokeChanged = oldAxis.ticks?.stroke !== newAxis.ticks?.stroke;
    return strokeChanged || borderStrokeChanged || gridStrokeChanged || ticksStrokeChanged;
  });
  if (someAxisStyleHasChanged) return true;

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
  const scalesRef = useRef<Record<string, Scale>>({});
  const dataRef = useRef<uPlot.AlignedData>(EMPTY_DATA);
  const [ isEmpty, setIsEmpty ] = useState(true);
  const [ isReady, setIsReady ] = useState(false);
  const resize = useResize(chartDivRef);

  const fullOptions = useMemo(() => {
    return uPlot.assign({
      cursor: {
        bind: {
          dblclick: () => {
            // Reset zoom for every scale key.
            Object.values(scalesRef.current).forEach(scale => {
              scale.isZoomed = false;
            });
          },
        },
      },
      hooks: {
        destroy: [ () => setIsReady(false) ],
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
          const currentMax = scalesRef.current[scaleKey]?.max;
          const currentMin = scalesRef.current[scaleKey]?.min;
          // Top resolves to max vertical value, and top + height resolves to min vertical value.
          const maxPos = isVertical ? u.bbox.top : u.bbox.left + u.bbox.width;
          const minPos = isVertical ? u.bbox.top + u.bbox.height : u.bbox.left;
          const newMax = u.posToVal(maxPos, scaleKey);
          const newMin = u.posToVal(minPos, scaleKey);

          if (currentMax == null || currentMin == null) {
            scalesRef.current[scaleKey] = { isZoomed: false, max: newMax, min: newMin };
          } else {
            scalesRef.current[scaleKey] = {
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
   * Effect to handle uPlot recreate and redraw where we need to compare the
   * previous and the current uPlot options.
   */
  useEffect(() => {
    if (!chartDivRef.current) return;
    if (shouldRecreate(fullOptions, prevFullOptions)) {
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
        handleError(e, { ...UPLOT_ERROR, publicSubject: 'Unable to create uPlot.' });
      }
    } else if (shouldRedraw(fullOptions, prevFullOptions)) {
      try {
        chartRef?.current?.redraw();
      } catch (e) {
        handleError(e, { ...UPLOT_ERROR, publicSubject: 'Unable to redraw uPlot.' });
      }
    }
  }, [ fullOptions, prevFullOptions ]);

  /**
   * Update `isEmpty` state and `dataRef` when source data changes.
   */
  useEffect(() => {
    const alignedData = data as uPlot.AlignedData | undefined;

    /**
     * We keep a reference of `data` to avoid triggering the recreate
     * useEffect above whenever data changes, and still allow the
     * recreate to use the latest version of `data`.
     */
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
      try {
        const isZoomed = Object.values(scalesRef.current).some(i => i.isZoomed);
        chartRef.current?.setData(dataRef.current, !isZoomed);
      } catch (e) {
        handleError(e, { ...UPLOT_ERROR, publicSubject: 'Unable to update uPlot data.' });
      }
    }
  }, [ data, fullOptions.mode, isReady ]);

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
      try {
        chartRef.current?.addSeries(newSeriesMap[label].series);
      } catch (e) {
        handleError(e, {
          ...UPLOT_ERROR,
          publicSubject: `Unable to add series "${label}" to uPlot.`,
        });
      }
    });

    // Remove existing series that no longer exists in `options.series`.
    Object.keys(oldSeriesMap).forEach(label => {
      if (newSeriesMap[label]) return;
      try {
        chartRef.current?.delSeries(oldSeriesMap[label].index);
      } catch (e) {
        handleError(e, {
          ...UPLOT_ERROR,
          publicSubject: `Unable to delete series "${label}" from uPlot.`,
        });
      }
    });
  }, [ fullOptions.mode, fullOptions.series ]);

  /**
   * Resize the chart when resize events happen.
   */
  useEffect(() => {
    const [ width, height ] = [ resize.width, fullOptions.height ];

    // Invalid width or height.
    if (!width || !height) return;

    // No need to set size since the new sizes are the same as the previous size.
    if (width === chartRef.current?.width && height === chartRef.current?.height) return;

    try {
      chartRef.current?.setSize({ height, width });
    } catch (e) {
      handleError(e, {
        ...UPLOT_ERROR,
        publicSubject: `Unable to set uPlot to ${width} x ${height}`,
      });
    }
  }, [ fullOptions.height, resize ]);

  /**
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    try {
      const hasFocus = focusIndex !== undefined;
      chartRef.current?.setSeries(hasFocus ? focusIndex as number + 1 : null, { focus: hasFocus });
    } catch (e) {
      handleError(e, {
        ...UPLOT_ERROR,
        publicSubject: `Unable to focus on uPlot series with index ${focusIndex}.`,
      });
    }
  }, [ focusIndex ]);

  /**
   * Resync the chart when scroll events happen to correct the cursor position upon
   * a parent container scrolling.
   */
  useEffect(() => {
    const throttleFunc = throttle(SCROLL_THROTTLE_TIME, () => {
      try {
        chartRef.current?.syncRect();
      } catch (e) {
        handleError(e, { ...UPLOT_ERROR, publicSubject: 'Unable to resync uPlot.' });
      }
    });
    const handleScroll = () => throttleFunc();

    /**
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
