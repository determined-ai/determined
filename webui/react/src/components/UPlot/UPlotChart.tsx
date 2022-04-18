import { option } from 'fp-ts/lib/Option';
import React, { useEffect, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import useResize from 'hooks/useResize';
import Message, { MessageType } from 'shared/components/message';
import { isEqual } from 'utils/data';

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

  // if (oldOptions.key !== newOptions.key) return true;
  if (Object.keys(oldOptions).length !== Object.keys(newOptions).length) return true;

  if (oldOptions.title !== newOptions.title) return true;
  if (oldOptions.axes?.length !== newOptions.axes?.length) return true;

  if (oldOptions.axes?.length !== newOptions.axes?.length) return true;
  if (oldOptions.series?.length !== newOptions.series?.length) return true;
  if (oldOptions.scales?.y?.distr !== newOptions.scales?.y?.distr) return true;

  const oldOptionKeys = JSON.stringify(Object.keys(oldOptions));
  const newOptionKeys = JSON.stringify(Object.keys(newOptions));
  if (oldOptionKeys !== newOptionKeys) return true;

  const someAxisLabelHasChanged = oldOptions.axes?.some((oldAxis, seriesIdx) => {
    const newAxis = newOptions.axes?.[seriesIdx];
    return oldAxis.label !== newAxis?.label;
  });
  if (someAxisLabelHasChanged) return true;

  const someSeriesHasChanged = chart.series.some((chartSeries, seriesIdx) => {
    const oldSeries = oldOptions.series?.[seriesIdx];
    const newSeries = newOptions.series?.[seriesIdx];
    return (
      (newSeries?.show != null && chartSeries?.show !== newSeries?.show)
      || (oldSeries?.show != null && oldSeries?.show !== newSeries?.show)
      || (newSeries?.label != null && chartSeries?.label !== newSeries?.label)
      || (newSeries?.fill != null && chartSeries?.fill !== newSeries?.fill)
    );
  });
  if (someSeriesHasChanged) return true;

  return false;
};

const extendOptions = (
  width?: number,
  options: Partial<uPlot.Options> = {},
  zoom: ZoomScales = {},
  onReady?: (uPlot: uPlot) => void,
  onDestroy?: (uPlot: uPlot) => void,
): uPlot.Options => uPlot.assign({
  hooks: {
    destroy: [ onDestroy ],
    ready: [ onReady ],
    setScale: [ (uPlot: uPlot, scaleKey: string) => {
      if (![ 'x', 'y' ].includes(scaleKey)) return;

      const boundWidth = scaleKey === 'x' ? uPlot.bbox.width : 0;
      const boundHeight = scaleKey === 'x' ? 0 : uPlot.bbox.height;
      const currentMax: number | undefined = uPlot.posToVal(boundWidth, scaleKey);
      const currentMin: number | undefined = uPlot.posToVal(boundHeight, scaleKey);
      let max: number | undefined = zoom[scaleKey]?.max;
      let min: number | undefined = zoom[scaleKey]?.min;

      if (max == null || currentMax > max) max = currentMax;
      if (min == null || currentMin < min) min = currentMin;

      zoom[scaleKey] = {
        isZoomed: currentMax < max || currentMin > min,
        max,
        min,
      };
    } ],
  },
  width,
}, options ?? {}) as uPlot.Options;

// setScale: [
//   (uPlot: uPlot, scaleKey: string) => {
//     /**
//      * This fires *after* scale change
//      * This function attempts to detect whether update was due to a zoom.
//      * It does so by looking at whether the bounds curresponding to the
//      * chart viewport are smaller than the maximum previous extent of those
//      * bounds. If so, it is "zoomed". Sometimes setData can cause this to happen.
//      * In particular, the scales generated for two points may be smaller than
//      * for the first point alone. We try to provide some help here by saying
//      * that if there is only one data point and an xScale is provided in options,
//      * then don't reset the scale in setData. But the consumer of the component
//      * still would need to manually provide a scale with min and max
//      * centered around the point when- and only when- their component has one point
//      */
//     if (scaleKey !== 'x') return;

//     // prevMax/Min indicate the previous max/min extent of the data.
//     const prevMax = boundsRef.current[scaleKey]?.max;
//     const prevMin = boundsRef.current[scaleKey]?.min;

//     const bboxMax: number | undefined = uPlot.posToVal(
//       scaleKey === 'x' ? uPlot.bbox.width : 0,
//       scaleKey,
//     );
//     const bboxMin: number | undefined = uPlot.posToVal(
//       scaleKey === 'x' ? 0 : uPlot.bbox.height,
//       scaleKey,
//     );

//     // the new max/min extent of the data
//     const maxMax = Math.max(prevMax ?? Number.MIN_SAFE_INTEGER, bboxMax);
//     const minMin = Math.min(prevMin ?? Number.MAX_SAFE_INTEGER, bboxMin);

//     /**
//      * here we are cheating a bit by assuming that a zoom is smaller on both ends
//      * this is to get around the issue of calling it a zoom when setData
//      * causes the scale to go from [99.9, 100.1] to [100, 200]
//      */
//     const isZoomed = bboxMax < maxMax && bboxMin > minMin;

//     boundsRef.current[scaleKey] = {
//       isZoomed,
//       max: maxMax,
//       min: minMin,
//     };
//   },
// ],

const UPlotChart: React.FC<Props> = ({
  data,
  focusIndex,
  options,
  style,
  noDataMessage,
}: Props) => {
  const chartRef = useRef<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const zoomRef = useRef<ZoomScales>({});
  const optionsRef = useRef<uPlot.Options>(extendOptions(undefined, options));
  const dataRef = useRef<uPlot.AlignedData>(EMPTY_DATA);
  const [ isEmpty, setIsEmpty ] = useState(true);
  const [ isReady, setIsReady ] = useState(false);
  const resize = useResize(chartDivRef);

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
    const newOptions = extendOptions(
      chartDivRef.current?.offsetWidth,
      options,
      zoomRef.current,
      () => setIsReady(true),
      () => setIsReady(false),
    );
    const recreate = shouldRecreate(chartRef.current, optionsRef.current, newOptions);
    console.log('recreate?', recreate);
    if (chartDivRef.current && shouldRecreate(chartRef.current, optionsRef.current, newOptions)) {
      optionsRef.current = newOptions;
      chartRef?.current?.destroy();
      chartRef.current = undefined;

      try {
        chartRef.current = new uPlot(optionsRef.current, dataRef.current, chartDivRef.current);
      } catch (e) {
        // Something happened during uPlot creation, setting as "no data" for now.
        setIsEmpty(true);
      }
    }
  }, [ options ]);

  /**
   * Update `isEmpty` state and `dataRef` when source data changes.
   */
  useEffect(() => {
    const alignedData = data as uPlot.AlignedData | undefined;

    if (optionsRef.current.mode === 2 && alignedData) {
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
      const isZoomed = Object.values(zoomRef.current).some(i => i.isZoomed === true);
      chartRef.current?.setData(dataRef.current, !isZoomed);
    }
  }, [ data, isReady ]);

  /**
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    if (!chartRef.current) return;
    const hasFocus = focusIndex !== undefined;
    chartRef.current.setSeries(hasFocus ? focusIndex as number + 1 : null, { focus: hasFocus });
  }, [ focusIndex ]);

  /**
   * When scales settings change in options, update chart scale.
   */
  useEffect(() => {
    if (!chartRef.current) return;

    const oldScaleKeys = Object.keys(optionsRef.current.scales ?? {}).sort();
    const newScales = options?.scales ?? {};
    const newScaleKeys = Object.keys(newScales).sort();
    if (isEqual(oldScaleKeys, newScaleKeys)) return;

    newScaleKeys.forEach(scaleKey => {
      const scales = newScales[scaleKey];
      if (scales.min != null && scales.max != null) {
        console.log('scales changed', scaleKey, scales);
        chartRef.current?.setScale(scaleKey, { max: scales.max, min: scales.min });
      }
    });
  }, [ options?.scales ]);

  /**
   * Resize the chart when resize events happen.
   */
  useEffect(() => {
    if (!chartRef.current || !resize.width || !resize.height) return;
    const [ width, height ] = [ resize.width, options?.height || chartRef.current.height ];

    if (chartRef.current.width === width && chartRef.current.height === height) return;

    chartRef.current.setSize({ height, width });
  }, [ options?.height, resize ]);

  /**
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
