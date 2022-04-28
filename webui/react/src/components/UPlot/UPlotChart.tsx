import React, { useEffect, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import Message, { MessageType } from 'components/Message';
import useResize from 'hooks/useResize';

import { FacetedData } from './types';

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

const UPlotChart: React.FC<Props> = ({ data, focusIndex, options, style }: Props) => {
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
          title="No data to plot."
          type={MessageType.Empty}
        />
      )}
    </div>
  );
};

export default UPlotChart;
