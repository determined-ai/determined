import React, { useCallback, useEffect, useRef, useState } from 'react';
import uPlot, { Options } from 'uplot';

import 'uplot/dist/uPlot.min.css';
import useResize from 'hooks/useResize';
import { distance } from 'utils/chart';

import css from './LearningCurveChart.module.scss';

interface Props {
  data: (number | null)[][];
  focusedTrialId?: number;
  onTrialFocus?: (trialId: number | null) => void;
  trialIds: number[];
  xValues: number[];
}

const CHART_HEIGHT = 400;
const CANVAS_CSS_RATIO = 2;
const FOCUS_MIN_DISTANCE = 30;
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

const LearningCurveChart: React.FC<Props> = ({
  data,
  focusedTrialId,
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
    if (!tooltipRef.current || !trialIdRef.current ||
        !batchesRef.current || !metricValueRef.current) return position;

    const maxIdx = xValues.length;
    const idx = plot.posToIdx(mouseLeft);

    // Find the nearest series and data point based on cursor position.
    let valDistance = Number.MAX_VALUE;
    let closestSeriesIdx = -1;
    let [ closestX, closestY ] = [ -1, -1 ];
    let [ closestXValue, closestValue ] = [ -1, -1 ];

    data.forEach((series, index) => {
      let idxDistance = 0;
      let searchLeft = true;
      let searchRight = true;

      // TODO: Optimize and refactor.
      while (searchLeft || searchRight) {
        const leftIdx = idx - idxDistance;
        const rightIdx = idx + idxDistance;
        if (leftIdx >= 0) {
          const leftXVal = xValues[leftIdx];
          const leftVal = series[leftIdx];
          if (leftVal != null) {
            const x = plot.valToPos(leftXVal, 'x');
            const y = plot.valToPos(leftVal, 'metric');
            const dist = distance(x, y, mouseLeft, mouseTop);
            if (dist > FOCUS_MIN_DISTANCE) {
              searchLeft = false;
            } else if (dist < valDistance) {
              valDistance = dist;
              closestSeriesIdx = index;
              closestX = x;
              closestY = y;
              closestXValue = leftXVal;
              closestValue = leftVal;
            }
          }
        } else {
          searchLeft = false;
        }
        if (rightIdx < maxIdx) {
          const rightXVal = xValues[rightIdx];
          const rightVal = series[rightIdx];
          if (rightVal != null) {
            const x = plot.valToPos(rightXVal, 'x');
            const y = plot.valToPos(rightVal, 'metric');
            const dist = distance(x, y, mouseLeft, mouseTop);
            if (dist > FOCUS_MIN_DISTANCE) {
              searchRight = false;
            } else if (dist < valDistance) {
              valDistance = dist;
              closestSeriesIdx = index;
              closestX = x;
              closestY = y;
              closestXValue = rightXVal;
              closestValue = rightVal;
            }
          }
        } else {
          searchRight = false;
        }
        idxDistance++;
      }
    });

    // Focus or remove focus series.
    if (closestSeriesIdx === -1) {
      plot.setSeries(null as unknown as number, { focus: false });
      if (onTrialFocus) onTrialFocus(null);
    } else {
      plot.setSeries(closestSeriesIdx + 1, { focus: true });
      if (onTrialFocus) onTrialFocus(trialIds[closestSeriesIdx]);
    }

    /*
     * Disable focus on individual data point.
     * uPlot picks the nearest point based on the X axis to focus on
     * and not the nearest point based on the cursor position.
     * Disable
     */
    plot.cursor.dataIdx = (): number => null as unknown as number;

    if (closestSeriesIdx !== -1) {
      const x = closestX + plot.bbox.left / CANVAS_CSS_RATIO;
      const y = closestY + plot.bbox.top / CANVAS_CSS_RATIO;
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
      trialIdRef.current.innerText = trialIds[closestSeriesIdx].toString();
      batchesRef.current.innerText = closestXValue.toString();
      metricValueRef.current.innerText = closestValue.toString();
    } else {
      tooltipRef.current.style.display = 'none';
    }

    return position;
  }, [ data, onTrialFocus, tooltipRef, trialIdRef, trialIds, xValues ]);

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
          stroke: 'rgba(50, 0, 150, 1.0)',
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
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ data, xValues ]);

  // Resize the chart when resize events detected.
  useEffect(() => {
    if (chart) chart.setSize({ height: CHART_HEIGHT, width: resize.width });
  }, [ chart, resize ]);

  // Focus on a trial series if provided.
  useEffect(() => focusOnTrial(), [ focusOnTrial ]);

  return (
    <div className={css.base}>
      <div ref={chartRef} onMouseLeave={handleMouseLeave} />
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
