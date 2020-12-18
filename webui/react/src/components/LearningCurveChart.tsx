import React, { useCallback, useEffect, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';
import uPlot, { Options } from 'uplot';

import 'uplot/dist/uPlot.min.css';
import useResize from 'hooks/useResize';
import { distance } from 'utils/chart';

import css from './LearningCurveChart.module.scss';

interface Props {
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialClick?: (event: React.MouseEvent, trialId: number) => void;
  onTrialFocus?: (trialId: number | null) => void;
  trialIds: number[];
  xValues: number[];
}

interface ClosestPoint {
  distance: number;
  seriesIdx?: number;
  value?: number;
  x?: number;
  xValue?: number;
  y?: number;
}

const CHART_HEIGHT = 400;
const CANVAS_CSS_RATIO = 2;
const FOCUS_MIN_DISTANCE = 30;
const SCROLL_THROTTLE_TIME = 500;
const UPLOT_OPTIONS = {
  axes: [
    {
      grid: { width: 1 },
      scale: 'x',
      side: 2,
    },
    {
      grid: { width: 1 },
      scale: 'metric',
      side: 3,
    },
  ],
  height: CHART_HEIGHT,
  legend: { show: false },
  scales: {
    metric: { auto: true, time: false },
    x: { time: false },
  },
  series: [ { label: 'batches' } ],
};

const findClosestPoint = (
  sharedData: {
    mouseLeft: number,
    mouseTop: number,
    plot: uPlot,
    series: (number | null)[],
    seriesIdx: number,
    startIdx: number,
    xValues: number[],
  },
  closestPoint: ClosestPoint,
  idxOffset: number,
): ClosestPoint => {
  const idx = sharedData.startIdx + idxOffset;
  if (idx < 0 || idx >= sharedData.xValues.length) return closestPoint;

  const xValue = sharedData.xValues[idx];
  const value = sharedData.series[idx];
  let updatedClosestPoint: ClosestPoint = { ...closestPoint };

  if (value != null) {
    const posX = sharedData.plot.valToPos(xValue, 'x');
    const posY = sharedData.plot.valToPos(value, 'metric');
    const dist = distance(posX, posY, sharedData.mouseLeft, sharedData.mouseTop);

    if (dist > FOCUS_MIN_DISTANCE) return closestPoint;
    if (dist < closestPoint.distance) {
      updatedClosestPoint = {
        distance: dist,
        seriesIdx: sharedData.seriesIdx,
        value,
        x: posX,
        xValue,
        y: posY,
      };
    }
  }

  if (idxOffset === 0) {
    const leftPoint = findClosestPoint(sharedData, updatedClosestPoint, -1);
    return findClosestPoint(sharedData, leftPoint, 1);
  }
  const nextIdxOffset = idxOffset + (idxOffset < 0 ? -1 : 1);
  return findClosestPoint(sharedData, updatedClosestPoint, nextIdxOffset);
};

const LearningCurveChart: React.FC<Props> = ({
  data,
  focusedTrialId,
  onTrialClick,
  onTrialFocus,
  trialIds,
  xValues,
}: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const trialIdRef = useRef<HTMLDivElement>(null);
  const batchesRef = useRef<HTMLDivElement>(null);
  const metricValueRef = useRef<HTMLDivElement>(null);
  const resize = useResize(chartRef);
  const [ chart, setChart ] = useState<uPlot>();
  const [ focusedPoint, setFocusedPoint ] = useState<ClosestPoint>();

  const focusOnTrial = useCallback(() => {
    if (!chart) return;

    let seriesIdx = -1;
    if (focusedTrialId && trialIds.includes(focusedTrialId)) {
      seriesIdx = trialIds.findIndex(id => id === focusedTrialId);
    }

    if (seriesIdx === -1) {
      chart.setSeries(null as unknown as number, { focus: false });
    } else {
      chart.setSeries(seriesIdx + 1, { focus: true });
    }
  }, [ chart, focusedTrialId, trialIds ]);

  const handleClick = useCallback((event: React.MouseEvent) => {
    if (!chart || !focusedPoint || focusedPoint.seriesIdx == null || !onTrialClick) return;
    onTrialClick(event, trialIds[focusedPoint.seriesIdx]);
  }, [ chart, focusedPoint, onTrialClick, trialIds ]);

  const handleMouseLeave = useCallback(() => {
    focusOnTrial();
    setTimeout(() => {
      if (tooltipRef.current) tooltipRef.current.style.display = 'none';
    }, 100);
  }, [ focusOnTrial ]);

  const handleCursorMove = useCallback((
    plot: uPlot,
    mouseLeft: number,
    mouseTop: number,
  ) => {
    const position = [ mouseLeft, mouseTop ];
    if (mouseLeft < 0 && mouseTop < 0) return position;
    if (!plot.data || plot.data.length === 0) return;
    if (!tooltipRef.current || !trialIdRef.current ||
        !batchesRef.current || !metricValueRef.current) return position;

    const localXValues = plot.data[0];
    const localData = plot.data.slice(1);
    const idx = plot.posToIdx(mouseLeft);

    // Find the nearest series and data point based on cursor position.
    let closestPoint: ClosestPoint = { distance: Number.MAX_VALUE };
    localData.forEach((series, index) => {
      closestPoint = findClosestPoint({
        mouseLeft,
        mouseTop,
        plot,
        series,
        seriesIdx: index,
        startIdx: idx,
        xValues: localXValues,
      }, closestPoint, 0);
    });
    setFocusedPoint(closestPoint);

    // Focus or remove focus series.
    if (closestPoint.seriesIdx == null) {
      plot.setSeries(null as unknown as number, { focus: false });
      if (onTrialFocus) onTrialFocus(null);
    } else {
      plot.setSeries(closestPoint.seriesIdx + 1, { focus: true });
      if (onTrialFocus) onTrialFocus(trialIds[closestPoint.seriesIdx]);
    }

    /*
     * Disable focus on individual data point.
     * uPlot picks the nearest point based on the X axis to focus on
     * and not the nearest point based on the cursor position.
     * Disable
     */
    plot.cursor.dataIdx = (): number => null as unknown as number;

    if (closestPoint.seriesIdx != null && closestPoint.x != null && closestPoint.y != null &&
        closestPoint.xValue != null && closestPoint.value != null) {
      const x = closestPoint.x + plot.bbox.left / CANVAS_CSS_RATIO;
      const y = closestPoint.y + plot.bbox.top / CANVAS_CSS_RATIO;
      const classes = [ css.tooltip ];

      /*
       * Place tooltip in the quadrant appropriate for where the cursor position is.
       * 1 - Bottom Right, 2 - Bottom Left, 3 - Top Right, 4 - Top Left
       */
      if (y > plot.bbox.height / 2 / CANVAS_CSS_RATIO) classes.push(css.top);
      if (x > plot.bbox.width / 2 / CANVAS_CSS_RATIO) classes.push(css.left);

      tooltipRef.current.style.display = 'block';
      tooltipRef.current.style.left = `${x}px`;
      tooltipRef.current.style.top = `${y}px`;
      tooltipRef.current.className = classes.join(' ');
      trialIdRef.current.innerText = trialIds[closestPoint.seriesIdx].toString();
      batchesRef.current.innerText = closestPoint.xValue.toString();
      metricValueRef.current.innerText = closestPoint.value.toString();
    } else {
      tooltipRef.current.style.display = 'none';
    }

    return position;
  }, [ onTrialFocus, tooltipRef, trialIdRef, trialIds ]);

  useEffect(() => {
    if (!chartRef.current) return;

    const options = uPlot.assign({}, UPLOT_OPTIONS, {
      cursor: { move: handleCursorMove },
      series: [
        { label: 'batches' },
        ...trialIds.map(trialId => ({
          label: `trial ${trialId}`,
          scale: 'metric',
          spanGaps: true,
          stroke: 'rgba(0, 155, 222, 1.0)',
          width: 1 / devicePixelRatio,
        })),
      ],
      width: chartRef.current.offsetWidth,
    }) as Options;

    const plotChart = new uPlot(options, [ xValues, ...data ], chartRef.current);
    setChart(plotChart);

    return () => {
      setChart(undefined);
      plotChart.destroy();
    };
  }, [ data, handleCursorMove, trialIds, xValues ]);

  // Focus on a trial series if provided.
  useEffect(() => focusOnTrial(), [ focusOnTrial ]);

  // Resize the chart when resize events happen.
  useEffect(() => {
    if (chart) chart.setSize({ height: CHART_HEIGHT, width: resize.width });
  }, [ chart, resize ]);

  /*
   * Resync the chart when scroll events happen to correct the cursor position upon
   * a parent container scrolling.
   */
  useEffect(() => {
    const throttleFunc = throttle(SCROLL_THROTTLE_TIME, () => {
      if (chart) chart.syncRect();
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
  }, [ chart ]);

  return (
    <div className={css.base}>
      <div ref={chartRef} onClick={handleClick} onMouseLeave={handleMouseLeave} />
      <div className={css.tooltip} ref={tooltipRef}>
        <div className={css.point} />
        <div className={css.box}>
          <div className={css.row}>
            <div>Trial Id:</div>
            <div ref={trialIdRef} />
          </div>
          <div className={css.row}>
            <div>Batches:</div>
            <div ref={batchesRef} />
          </div>
          <div className={css.row}>
            <div>Metric:</div>
            <div ref={metricValueRef} />
          </div>
        </div>
      </div>
    </div>
  );
};

export default LearningCurveChart;
