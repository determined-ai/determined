import React, { useEffect, useMemo, useRef } from 'react';
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

interface ScaleZoomData {
  isZoomed?: boolean;
  max?: number;
  min?: number;
}

const shouldUpdate = (
  prev: Partial<uPlot.Options> | undefined,
  next: Partial<uPlot.Options> | undefined,
  chart: uPlot | undefined,
): boolean => {
  if (!chart || !next) return false;
  if (!prev) return true;
  if (Object.keys(prev).length !== Object.keys(next).length) {
    return true;
  }
  if (prev.axes?.length !== next.axes?.length) {
    return true;
  }

  if (
    prev.axes?.some((prevAxis, seriesIdx) => {
      const nextAxis = next.axes?.[seriesIdx];
      return prevAxis.label !== nextAxis?.label;
    })
  ) {
    return true;
  }

  if (chart?.series?.length !== next.series?.length) {
    return true;
  }

  if (
    chart.series.some((chartSerie, seriesIdx) => {
      const nextSerie = next.series?.[seriesIdx];
      return (
        (nextSerie?.show != null && chartSerie?.show !== nextSerie?.show) ||
        (nextSerie?.label != null && chartSerie?.label !== nextSerie?.label)
      );
    })
  ) {
    return true;
  }

  return false;
};

const getNormalizedData = (data: AlignedData | FacetedData | undefined, options: uPlot.Options) => {
  if (!data || data.length < 2) return [ false, undefined ];

  // Is the chart aligned (eg. linear) or faceted (eg. scatter plot)?
  if (options?.mode === 2) {
    return [ true, data as AlignedData ];
  } else {
    // Figure out the lowest sized series data.
    const chartData = data as AlignedData;
    const minDataLength = chartData.reduce((acc: number, series: UPlotData[]) => {
      return Math.min(acc, series.length);
    }, Number.MAX_SAFE_INTEGER);

    // Making sure the X series and all the other series data are the same length;
    const trimmedData = chartData.map(series => series.slice(0, minDataLength));

    // Checking to make sure the X series has some data.
    const hasXValues = (trimmedData?.[0]?.length ?? 0) !== 0;

    return [ hasXValues, trimmedData as unknown as AlignedData ];
  }
};
const SCROLL_THROTTLE_TIME = 500;

const UPlotChart: React.FC<Props> = ({ data, focusIndex, options, style }: Props) => {
  const chartRef = useRef<uPlot>();
  const chartDivRef = useRef<HTMLDivElement>(null);
  const scalesZoomData = useRef<Record<string, ScaleZoomData>>({});

  const getAugmentedOptions = (options: Partial<uPlot.Options> | undefined) => uPlot.assign(
    {

      hooks: {
        setScale: [ (uPlot: uPlot, scaleKey: string) => {
          if (![ 'x', 'y' ].includes(scaleKey)) return;

          const currentMax: number|undefined =
            uPlot.posToVal(scaleKey === 'x' ? uPlot.bbox.width : 0, scaleKey);
          const currentMin: number|undefined =
            uPlot.posToVal(scaleKey === 'x' ? 0 : uPlot.bbox.height, scaleKey);
          let max: number|undefined = scalesZoomData.current[scaleKey]?.max;
          let min: number|undefined = scalesZoomData.current[scaleKey]?.min;

          if (max == null || currentMax > max) max = currentMax;
          if (min == null || currentMin < min) min = currentMin;

          scalesZoomData.current[scaleKey] = {
            isZoomed: currentMax < max || currentMin > min,
            max,
            min,
          };
        } ],
      },
      width: chartDivRef.current?.offsetWidth,
    },
    options || {},
  ) as uPlot.Options;

  const optionsRef = useRef<uPlot.Options>(getAugmentedOptions(options));

  const [ hasData, normalizedData ] = useMemo(
    () => getNormalizedData(data, optionsRef.current)
    , [ data ],
  );

  /*
   * Chart mount and dismount.
   */
  useEffect(() => {
    if (!chartDivRef.current) return;
    scalesZoomData.current = {};
    const data = [ [], [ [] ] ] as unknown as uPlot.AlignedData;
    if (!chartRef?.current) {
      chartRef.current = new uPlot(optionsRef.current, data, chartDivRef.current);
    }
    return () => {
      chartRef?.current?.destroy();
      chartRef.current = undefined;
    };
  }, [ ]);

  useEffect(() => {
    if (!chartDivRef.current) return;
    if (shouldUpdate(optionsRef.current, options, chartRef.current)) {
      console.log('create');
      chartRef.current?.destroy();
      chartRef.current = undefined;
      const newOptions = uPlot.assign(optionsRef.current, options || {}) as uPlot.Options;
      chartRef.current = new uPlot(
        newOptions,
        normalizedData as AlignedData,
        chartDivRef.current,
      );
    }
    return () => {
      if (options) optionsRef.current = options as uPlot.Options;
    };

  }, [ options, normalizedData ]);

  /*
   * Chart data when data changes.
   */
  useEffect(() => {
    if (!chartRef.current || !normalizedData) return;
    const isZoomed = Object.values(scalesZoomData.current).some(i => i.isZoomed === true);
    chartRef.current.setData(normalizedData as AlignedData, !isZoomed);
  }, [ chartRef, hasData, normalizedData ]);
  /*
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    if (!chartRef.current) return;
    const hasFocus = focusIndex !== undefined;
    chartRef.current.setSeries(hasFocus ? focusIndex as number + 1 : null, { focus: hasFocus });'';
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
    <div ref={chartDivRef} style={{ ...style }}>
      {!hasData && (
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
