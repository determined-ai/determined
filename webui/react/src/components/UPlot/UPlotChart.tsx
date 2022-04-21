import React, { useEffect, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import Message, { MessageType } from 'components/Message';
import useResize from 'hooks/useResize';

import { FacetedData, UPlotData } from './types';

export interface Options extends Omit<uPlot.Options, 'width'> {
  width?: number;
}

interface Props {
  data?: AlignedData | FacetedData;
  focusIndex?: number;
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

  if (oldOptions.axes?.length !== newOptions.axes?.length) return true;
  if (oldOptions.series?.length !== newOptions.series?.length) return true;

  const oldOptionKeys = JSON.stringify(Object.keys(oldOptions));
  const newOptionKeys = JSON.stringify(Object.keys(newOptions));
  if (oldOptionKeys !== newOptionKeys) return true;

  const someAxisLabelHasChanged = oldOptions.axes?.some((prevAxis, seriesIdx) => {
    const nextAxis = newOptions.axes?.[seriesIdx];
    return prevAxis.label !== nextAxis?.label;
  });
  if (someAxisLabelHasChanged) return true;

  const someSeriesHasChanged = chart.series.some((chartSerie, seriesIdx) => {
    const nextSerie = newOptions.series?.[seriesIdx];
    return (
      (nextSerie?.show != null && chartSerie?.show !== nextSerie?.show)
      || (nextSerie?.label != null && chartSerie?.label !== nextSerie?.label)
      || (nextSerie?.fill != null && chartSerie?.fill !== nextSerie?.fill)
    );
  });
  if(someSeriesHasChanged) return true;

  return false;
};

const extendOptions = (
  width?: number,
  options: Partial<uPlot.Options> = {},
  zoom: ZoomScales = {},
): uPlot.Options => uPlot.assign({
  hooks: {
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

const UPlotChart: React.FC<Props> = ({ data, focusIndex, options, style }: Props) => {
  const chartRef = useRef<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const zoomRef = useRef<ZoomScales>({});
  const optionsRef = useRef<uPlot.Options>(extendOptions(undefined, options));
  const dataRef = useRef<uPlot.AlignedData>(EMPTY_DATA);
  const [ isEmpty, setIsEmpty ] = useState(true);

  /**
   * Ensure that the chart is cleaned up during unmount if applicable.
   */
  useEffect(() => {
    console.log('mounting');
    return () => {
      console.log('destroying');
      chartRef?.current?.destroy();
      chartRef.current = undefined;
    };
  }, []);

  /**
   * Update `isEmpty` state and `dataRef` when source data changes.
   */
  useEffect(() => {
    let isEmpty = true;

    if (optionsRef.current.mode === 2) {
      dataRef.current = data as uPlot.AlignedData;
      isEmpty = false;
    } else if (data && data.length >= 2) {
      // Figure out the lowest sized series data.
      const alignedData = data as uPlot.AlignedData;
      const minDataLength = alignedData.reduce((acc: number, series: UPlotData[]) => {
        return Math.min(acc, series.length);
      }, Number.MAX_SAFE_INTEGER);

      // Making sure the X series and all the other series data are the same length;
      const trimmedData = alignedData.map((series) => series.slice(0, minDataLength));
      dataRef.current = trimmedData as unknown as uPlot.AlignedData;

      // Checking to make sure the X series has some data.
      isEmpty = (trimmedData?.[0]?.length ?? 0) === 0;
    } else {
      dataRef.current = EMPTY_DATA;
    }

    // Update chart with data changes if chart already exists.
    if (chartRef.current) {
      const isZoomed = Object.values(zoomRef.current).some(i => i.isZoomed === true);
      console.log('setData', dataRef.current, 'isZoomed', isZoomed);
      chartRef.current.setData(dataRef.current, !isZoomed);
    }

    setIsEmpty(isEmpty);
  }, [ data ]);

  /**
   * Recreate chart if the new `options` prop changes require it.
   */
  useEffect(() => {
    const newOptions = extendOptions(chartDivRef.current?.offsetWidth, options, zoomRef.current);
    console.log('should recreate', shouldRecreate(chartRef.current, optionsRef.current, newOptions));
    if (chartDivRef.current && shouldRecreate(chartRef.current, optionsRef.current, newOptions)) {
      console.log('recreating', dataRef.current);
      optionsRef.current = newOptions;
      chartRef?.current?.destroy();
      chartRef.current = new uPlot(optionsRef.current, dataRef.current, chartDivRef.current);
    }
  }, [ options ]);

  /**
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    if (!chartRef.current) return;
    const hasFocus = focusIndex !== undefined;
    chartRef.current.setSeries(hasFocus ? focusIndex as number + 1 : null, { focus: hasFocus });
  }, [ focusIndex ]);

  /**
   * Resize the chart when resize events happen.
   */
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chartRef.current) return;
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
          title="No data to plot."
          type={MessageType.Empty}
        />
      )}
    </div>
  );
};

export default UPlotChart;
