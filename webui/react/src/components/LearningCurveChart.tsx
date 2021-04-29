import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import uPlot, { AlignedData } from 'uplot';

import UPlotChart, { Options } from 'components/UPlotChart';
import { closestPointPlugin } from 'components/UPlotChart/closestPointPlugin';
import { MetricName } from 'types';
import { distance } from 'utils/chart';
import { glasbeyColor } from 'utils/color';
import { metricNameToStr } from 'utils/string';

import css from './LearningCurveChart.module.scss';

interface Props {
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialClick?: (event: React.MouseEvent, trialId: number) => void;
  onTrialFocus?: (trialId: number | null) => void;
  selectedMetric: MetricName;
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

interface Point {
  x: number;
  y: number;
}

const CHART_HEIGHT = 400;
const SERIES_WIDTH = 3;
const SERIES_UNFOCUSED_ALPHA = 0.2;
const FOCUS_MIN_DISTANCE = 30;
const MOUSE_CLICK_THRESHOLD = 5;
const SCROLL_THROTTLE_TIME = 500;

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
  selectedMetric,
  trialIds,
  xValues,
}: Props) => {
  const chart = useRef<uPlot>();

  // const handleClick = useCallback((event: React.MouseEvent) => {
  //   if (!chart || !mouseDownPoint) return;
  //
  //   /*
  //    * Make sure the mouse down and mouse up distance is fairly close
  //    * to be considered a click instead of a drag movement for chart zoom.
  //    */
  //   const dist = distance(event.clientX, event.clientY, mouseDownPoint?.x, mouseDownPoint?.y);
  //   if (dist < MOUSE_CLICK_THRESHOLD) {
  //     if (focusedPoint && focusedPoint.seriesIdx != null && onTrialClick) {
  //       onTrialClick(event, trialIds[focusedPoint.seriesIdx]);
  //     }
  //     setShowZoomOutTip(false);
  //   } else {
  //     setShowZoomOutTip(true);
  //   }
  //
  //   setMouseDownPoint(undefined);
  // }, [ chart, focusedPoint, mouseDownPoint, onTrialClick, trialIds ]);
  //
  // const handleMouseLeave = useCallback(() => {
  //   focusOnTrial();
  //   setTimeout(() => {
  //     if (tooltipRef.current) tooltipRef.current.style.display = 'none';
  //     if (onTrialFocus) onTrialFocus(null);
  //   }, 100);
  // }, [ focusOnTrial, onTrialFocus ]);

  // const handleCursorMove = useCallback((
  //   plot: uPlot,
  //   mouseLeft: number,
  //   mouseTop: number,
  // ) => {
  //   const position = [ mouseLeft, mouseTop ];
  //   if (mouseLeft < 0 && mouseTop < 0) return position;
  //   if (!plot.data || plot.data.length === 0) return;
  //   if (!tooltipRef.current || !pointRef.current || !trialIdRef.current ||
  //       !batchesRef.current || !metricValueRef.current) return position;
  //
  //   const localXValues = plot.data[0];
  //   const localData = plot.data.slice(1);
  //   const idx = plot.posToIdx(mouseLeft);
  //
  //   // Find the nearest series and data point based on cursor position.
  //   let closestPoint: ClosestPoint = { distance: Number.MAX_VALUE };
  //   localData.forEach((series, index) => {
  //     closestPoint = findClosestPoint({
  //       mouseLeft,
  //       mouseTop,
  //       plot,
  //       series,
  //       seriesIdx: index,
  //       startIdx: idx,
  //       xValues: localXValues,
  //     }, closestPoint, 0);
  //   });
  //   setFocusedPoint(closestPoint);
  //
  //   // Focus or remove focus series.
  //   if (closestPoint.seriesIdx == null) {
  //     plot.setSeries(null as unknown as number, { focus: false });
  //     if (onTrialFocus) onTrialFocus(null);
  //   } else {
  //     plot.setSeries(closestPoint.seriesIdx + 1, { focus: true });
  //     if (onTrialFocus) onTrialFocus(trialIds[closestPoint.seriesIdx]);
  //   }
  //
  //   /*
  //    * Disable focus on individual data point.
  //    * uPlot picks the nearest point based on the X axis to focus on
  //    * and not the nearest point based on the cursor position.
  //    * Disable
  //    */
  //   plot.cursor.dataIdx = (): number => null as unknown as number;
  //
  //   if (closestPoint.seriesIdx != null && closestPoint.x != null && closestPoint.y != null &&
  //       closestPoint.xValue != null && closestPoint.value != null) {
  //     const scale = window.devicePixelRatio;
  //     const x = closestPoint.x + plot.bbox.left / scale;
  //     const y = closestPoint.y + plot.bbox.top / scale;
  //     const classes = [ css.tooltip ];
  //
  //     /*
  //      * Place tooltip in the quadrant appropriate for where the cursor position is.
  //      * 1 - Bottom Right, 2 - Bottom Left, 3 - Top Right, 4 - Top Left
  //      */
  //     if (y > plot.bbox.height / 2 / scale) classes.push(css.top);
  //     if (x > plot.bbox.width / 2 / scale) classes.push(css.left);
  //
  //     tooltipRef.current.style.display = 'block';
  //     tooltipRef.current.style.left = `${x}px`;
  //     tooltipRef.current.style.top = `${y}px`;
  //     tooltipRef.current.className = classes.join(' ');
  //     pointRef.current.style.backgroundColor = glasbeyColor(closestPoint.seriesIdx);
  //     trialIdRef.current.innerText = trialIds[closestPoint.seriesIdx].toString();
  //     batchesRef.current.innerText = closestPoint.xValue.toString();
  //     metricValueRef.current.innerText = closestPoint.value.toString();
  //   } else {
  //     tooltipRef.current.style.display = 'none';
  //   }
  //
  //   return position;
  // }, [ onTrialFocus, tooltipRef, trialIdRef, trialIds ]);

  const chartData: AlignedData = useMemo(() => {
    return [
      [ 10, 20, 30, 40, 50, 60, 70, 80, 90, 100 ],
      [ 1, 3, 2, 1, 3, 2, 1, 3, 2, 1 ],
    ];
    return [ xValues, ...data ];
  }, [ data, xValues ]);
  const chartOptions: Options = useMemo(() => {
    return {
      axes: [
        {
          grid: { width: 1 },
          label: 'Batches Processed',
          scale: 'x',
          side: 2,
        },
        {
          grid: { width: 1 },
          label: metricNameToStr(selectedMetric),
          scale: 'metric',
          side: 3,
        },
      ],
      focus: { alpha: SERIES_UNFOCUSED_ALPHA },
      height: CHART_HEIGHT,
      legend: { show: false },
      plugins: [ closestPointPlugin({ distInPx: 30, yScale: 'metric' }) ],
      scales: {
        metric: { time: false },
        x: { time: false },
      },
      series: [
        { label: 'batches' },
        {
          label: 'metric',
          scale: 'metric',
          stroke: glasbeyColor(0),
          width: SERIES_WIDTH / window.devicePixelRatio,
        },
        // ...trialIds.map((trialId, index) => ({
        //   label: `trial ${trialId}`,
        //   scale: 'metric',
        //   spanGaps: true,
        //   stroke: glasbeyColor(index),
        //   width: SERIES_WIDTH / window.devicePixelRatio,
        // })),
      ],
    };
  }, [ selectedMetric, trialIds ]);

  // Focus on a trial series if provided.
  const focusOnTrial = useCallback(() => {
    if (!chart.current) return;

    let seriesIdx = -1;
    if (focusedTrialId && trialIds.includes(focusedTrialId)) {
      seriesIdx = trialIds.findIndex(id => id === focusedTrialId);
    }

    if (seriesIdx === -1) {
      chart.current.setSeries(null as unknown as number, { focus: false });
    } else {
      chart.current.setSeries(seriesIdx + 1, { focus: true });
    }
  }, [ chart, focusedTrialId, trialIds ]);
  useEffect(() => focusOnTrial(), [ focusOnTrial ]);

  return (
    <div className={css.base}>
      <UPlotChart data={chartData} options={chartOptions} ref={chart} />
    </div>
  );
};

export default LearningCurveChart;
