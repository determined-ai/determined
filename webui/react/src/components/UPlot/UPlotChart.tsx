import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import { useStore } from 'contexts/Store';
import useResize from 'hooks/useResize';
import Message, { MessageType } from 'shared/components/Message';
import usePrevious from 'shared/hooks/usePrevious';
import { DarkLight } from 'shared/themes';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

import { useSyncableBounds } from './SyncableBounds';
import { FacetedData } from './types';
import css from './UPlotChart.module.scss';

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

  const someScaleHasChanged = Object.entries(next.scales ?? {}).some(([scaleKey, nextScale]) => {
    const prevScale = prev?.scales?.[scaleKey];
    return prevScale?.distr !== nextScale?.distr;
  });

  if (someScaleHasChanged) return true;

  const someAxisHasChanged = prev.axes?.some((prevAxis, seriesIdx) => {
    const nextAxis = next.axes?.[seriesIdx];
    return (
      prevAxis.label !== nextAxis?.label ||
      prevAxis.stroke !== nextAxis?.stroke ||
      prevAxis.scale !== nextAxis?.scale
    );
  });
  if (someAxisHasChanged) return true;

  const someSeriesHasChanged = chart.series.some((chartSerie, seriesIdx) => {
    const nextSerie = next.series?.[seriesIdx];
    const prevSerie = prev.series?.[seriesIdx];
    return (
      (nextSerie?.show != null && chartSerie?.show !== nextSerie?.show) ||
      (prevSerie?.show != null && prevSerie?.show !== nextSerie?.show) ||
      (nextSerie?.label != null && chartSerie?.label !== nextSerie?.label) ||
      (nextSerie?.stroke != null && chartSerie?.stroke !== nextSerie?.stroke) ||
      (nextSerie?.paths != null && chartSerie?.paths !== nextSerie?.paths) ||
      (nextSerie?.fill != null && chartSerie?.fill !== nextSerie?.fill)
    );
  });
  if (someSeriesHasChanged) return true;

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
  const [isReady, setIsReady] = useState(false);
  const classes = [css.base];

  const { ui } = useStore();
  const { zoomed, boundsOptions, setZoomed } = useSyncableBounds();

  const hasData = data && data.length > 1 && (options?.mode === 2 || data?.[0]?.length);

  if (ui.darkLight === DarkLight.Dark) classes.push(css.dark);

  const extendedOptions = useMemo(() => {
    const extended: Partial<uPlot.Options> = uPlot.assign(
      {
        hooks: {
          destroy: [() => setIsReady(false), () => setZoomed(false)],
          ready: [() => setIsReady(true)],
        },
        width: chartDivRef.current?.offsetWidth,
      },
      boundsOptions || {},
      options || {},
    );

    // Override chart support colors to match theme.
    if (ui.theme && extended.axes) {
      const borderColor = ui.theme.surfaceBorderWeak;
      const labelColor = ui.theme.surfaceOn;
      extended.axes = extended.axes.map((axis) => {
        return {
          ...axis,
          border: { stroke: borderColor },
          grid: { stroke: borderColor },
          stroke: labelColor,
          ticks: { stroke: borderColor },
        };
      });
    }

    return extended as uPlot.Options;
  }, [boundsOptions, options, setZoomed, ui.theme]);

  const previousOptions = usePrevious(extendedOptions, undefined);

  useEffect(() => {
    return () => {
      chartRef?.current?.destroy();
      chartRef.current = undefined;
    };
  }, []);

  useEffect(() => {
    if (!chartDivRef.current) return;
    if (shouldRecreate(previousOptions, extendedOptions, chartRef.current)) {
      chartRef.current?.destroy();
      chartRef.current = undefined;
      try {
        if (extendedOptions?.mode === 2 || extendedOptions.series.length === data?.length) {
          chartRef.current = new uPlot(extendedOptions, data as AlignedData, chartDivRef.current);
        }
      } catch (e) {
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
        if (chartRef.current && isReady) {
          chartRef.current.setData(data as AlignedData, !zoomed);
        }
      } catch (e) {
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
  }, [data, extendedOptions, isReady, previousOptions, title, zoomed]);

  /**
   * When a focus index is provided, highlight applicable series.
   */
  useEffect(() => {
    if (!chartRef.current) return;
    const hasFocus = focusIndex !== undefined;
    chartRef.current.setSeries(hasFocus ? (focusIndex as number) + 1 : null, { focus: hasFocus });
  }, [focusIndex]);

  /*
   * Resize the chart when resize events happen.
   */
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chartRef.current) return;
    const [width, height] = [resize.width, options?.height || chartRef.current.height];
    if (chartRef.current.width === width && chartRef.current.height === height) return;
    chartRef.current.setSize({ height, width });
  }, [options?.height, resize]);

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
    <div className={classes.join(' ')} ref={chartDivRef} style={style}>
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
