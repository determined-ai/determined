import { DownloadOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import React, { RefObject, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { AlignedData } from 'uplot';

import Spinner from 'components/Spinner';
import usePrevious from 'hooks/usePrevious';
import useResize from 'hooks/useResize';
import useUI from 'stores/contexts/UI';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { DarkLight } from 'utils/themes';

import { useChartSync } from './SyncProvider';
import { FacetedData } from './types';
import css from './UPlotChart.module.scss';

export interface Options extends Omit<uPlot.Options, 'width'> {
  key?: number;
  width?: number;
}

interface Props {
  allowDownload?: boolean;
  data?: AlignedData | FacetedData;
  experimentId?: number;
  isLoading?: boolean;
  options?: Partial<Options>;
  style?: React.CSSProperties;
}

const SCROLL_THROTTLE_TIME = 500;

const shouldRecreate = (
  prev: Partial<Options> | undefined,
  next: Partial<Options> | undefined,
): boolean => {
  if (!next) return false;
  if (!prev) return true;
  if (prev === next) return false;
  if (prev.key !== next.key) return true;
  if (Object.keys(prev).length !== Object.keys(next).length) return true;

  if (prev.axes?.length !== next.axes?.length) return true;

  if (prev?.series?.length !== next.series?.length) return true;

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

  const someSeriesHasChanged = prev.series?.some((prevSerie, seriesIdx) => {
    const nextSerie = next.series?.[seriesIdx];

    return (
      (nextSerie?.label != null && prevSerie?.label !== nextSerie?.label) ||
      (prevSerie?.stroke != null && prevSerie?.stroke !== nextSerie?.stroke) ||
      (nextSerie?.paths != null && prevSerie?.paths !== nextSerie?.paths) ||
      (nextSerie?.fill != null && prevSerie?.fill !== nextSerie?.fill) ||
      prevSerie?.points?.show !== nextSerie?.points?.show
    );
  });
  if (someSeriesHasChanged) return true;

  return false;
};
type ChartType = 'Line' | 'Scatter';

const UPlotChart: React.FC<Props> = ({
  allowDownload,
  data,
  isLoading,
  options,
  style,
  experimentId,
}: Props) => {
  const chartRef = useRef<uPlot>();
  const [divHeight, setDivHeight] = useState((options?.height ?? 300) + 20);
  const chartDivRef = useRef<HTMLDivElement>(null);
  const classes = [css.base];

  const { ui } = useUI();
  const { options: syncOptions, syncService } = useChartSync();

  // line charts have their zoom state handled by `SyncProvider`, scatter charts do not.
  const chartType: ChartType = options?.mode === 2 ? 'Scatter' : 'Line';

  const hasData = data && data.length > 1 && (chartType === 'Scatter' || data?.[0]?.length);

  if (ui.darkLight === DarkLight.Dark) classes.push(css.dark);

  useEffect(() => {
    if (data !== undefined && chartType === 'Line')
      syncService.updateDataBounds(data as AlignedData);
  }, [syncService, chartType, data]);

  const extendedOptions = useMemo(() => {
    const extended: Partial<uPlot.Options> = uPlot.assign(
      {
        width: chartDivRef.current?.offsetWidth,
      },
      chartType === 'Line' ? syncOptions : {},
      options ?? {},
    );

    if (chartType === 'Line') {
      const activeBounds = syncService.activeBounds.get();
      if (activeBounds) {
        const { min, max } = activeBounds;
        const xScale = extended.scales?.x;
        if (xScale) {
          xScale.max = max;
          xScale.min = min;
        }
      }
    }

    // Override chart support colors to match theme.
    if (ui.theme && extended.axes) {
      const borderColor = ui.theme.surfaceBorderWeak;
      const labelColor = ui.theme.surfaceOn;
      extended.axes = extended.axes.map((axis) => {
        return {
          ...axis,
          border: { stroke: borderColor },
          grid: { ...axis.grid, stroke: borderColor },
          stroke: labelColor,
          ticks: { ...axis.ticks, stroke: borderColor },
        };
      });
    }

    return extended as uPlot.Options;
  }, [options, ui.theme, chartType, syncOptions, syncService]);

  const previousOptions = usePrevious(extendedOptions, undefined);

  useEffect(() => {
    return () => {
      chartRef?.current?.destroy();
      chartRef.current = undefined;
    };
  }, []);

  useEffect(() => {
    if (!chartDivRef.current) return;
    if (!hasData) {
      chartRef.current?.destroy();
      chartRef.current = undefined;
      return;
    }
    if (!chartRef.current || shouldRecreate(previousOptions, extendedOptions)) {
      chartRef.current?.destroy();
      chartRef.current = undefined;
      try {
        if (chartType === 'Scatter' || extendedOptions.series.length === data?.length) {
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
        chartRef.current?.setData(data as AlignedData, chartType === 'Scatter');
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
  }, [data, hasData, extendedOptions, previousOptions, chartType]);

  useEffect(() => {
    extendedOptions.series.forEach((ser, i) => {
      const chartSer = chartRef.current?.series?.[i];
      if (chartSer && chartSer.show !== ser?.show)
        chartRef.current?.setSeries(i, { show: ser.show }, false);
    });
  }, [extendedOptions.series]);

  /*
   * Resize the chart when resize events happen.
   */
  const resize = useResize(chartDivRef);
  useEffect(() => {
    if (!chartRef.current) return;
    const [width, height] = [resize.width, options?.height || chartRef.current.height];
    if (chartRef.current.width === width && chartRef.current.height === height) return;
    chartRef.current.setSize({ height, width });
    const container = chartDivRef.current;
    if (container && height) setDivHeight(height);
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
    <div className={classes.join(' ')} ref={chartDivRef} style={{ ...style, height: divHeight }}>
      {allowDownload && <DownloadButton containerRef={chartDivRef} experimentId={experimentId} />}
      {!hasData && !isLoading && (
        <div className={css.chartEmpty}>
          <span>No data to plot.</span>
        </div>
      )}
      {isLoading && <Spinner spinning tip="Loading chart data..." />}
    </div>
  );
};

export default UPlotChart;

const DownloadButton = ({
  containerRef,
  experimentId,
}: {
  containerRef: RefObject<HTMLDivElement>;
  experimentId?: number;
}) => {
  const downloadUrl = useRef<string>();
  const downloadNode = useRef<HTMLAnchorElement>(null);
  const fileName = useMemo(
    () => (experimentId ? `chart-trial-${experimentId}.png` : 'chart.png'),
    [experimentId],
  );

  useEffect(() => {
    return () => {
      if (downloadUrl.current) URL.revokeObjectURL(downloadUrl.current);
    };
  }, []);

  const handleDownloadClick = useCallback(() => {
    if (downloadUrl.current) URL.revokeObjectURL(downloadUrl.current);
    const canvas = containerRef.current?.querySelector('canvas');
    const url = canvas?.toDataURL('image/png');
    if (url && downloadNode.current) {
      downloadNode.current.href = url;
      downloadNode.current.click();
    }
    downloadUrl.current = url;
  }, [containerRef]);

  return (
    <Tooltip className={css.download} title="Download Chart">
      <DownloadOutlined onClick={handleDownloadClick} />
      {/* this is an invisible button to programatically download the image file */}
      <a
        aria-disabled
        className={css.invisibleLink}
        download={fileName}
        href={downloadUrl.current}
        ref={downloadNode}
      />
    </Tooltip>
  );
};
