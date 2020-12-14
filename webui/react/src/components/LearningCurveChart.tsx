import React, { useCallback, useEffect, useRef } from 'react';
import uPlot, { Cursor, Options } from 'uplot';

import 'uplot/dist/uPlot.min.css';
import { distance } from 'utils/chart';

import css from './LearningCurveChart.module.scss';

interface Props {
  data: (number | null)[][];
  trialIds: number[];
  xValues: number[];
}

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
  focus: { alpha: 0.3 },
  height: 400,
  legend: { show: false },
  scales: {
    metric: { auto: true, time: false },
    x: { time: false },
  },
  series: [ { label: 'batches' } ],
};

const FOCUS_MIN_DISTANCE = 30;

const LearningCurveChart: React.FC<Props> = ({ data, trialIds, xValues }: Props) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const trialIdRef = useRef<HTMLDivElement>(null);
  const batchesRef = useRef<HTMLDivElement>(null);
  const metricValueRef = useRef<HTMLDivElement>(null);

  const handleMouseLeave = useCallback(() => {
    return (plot: uPlot, target: HTMLElement, handler: Cursor.MouseListener) => {
      setTimeout(() => {
        if (tooltipRef.current) tooltipRef.current.style.display = 'none';
      }, 100);
      return handler;
    };
  }, []);

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

    let valDistance = Number.MAX_VALUE;
    let closestDataIdx = -1;
    let closestSeriesIdx = -1;
    let [ closestX, closestY ] = [ -1, -1 ];
    let [ closestXValue, closestValue ] = [ -1, -1 ];

    data.forEach((series, index) => {
      let idxDistance = 0;
      let searchLeft = true;
      let searchRight = true;

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
              closestDataIdx = leftIdx;
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
              closestDataIdx = leftIdx;
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

    plot.setSeries(closestSeriesIdx + 1, { focus: true });
    plot.cursor.dataIdx = (
      plot: uPlot,
      seriesIdx: number,
      closestIdx: number,
    ): number => {
      if (seriesIdx === closestSeriesIdx && closestIdx === closestDataIdx) {
        return closestDataIdx;
      }
      return -1;
    };

    if (closestSeriesIdx !== -1) {
      const [ offsetX, offsetY ] = [ plot.bbox.left / 2, plot.bbox.top / 2 ];
      tooltipRef.current.style.display = 'block';
      tooltipRef.current.style.left = `${closestX + offsetX}px`;
      tooltipRef.current.style.top = `${closestY + offsetY}px`;
      trialIdRef.current.innerText = trialIds[closestSeriesIdx].toString();
      batchesRef.current.innerText = closestXValue.toString();
      metricValueRef.current.innerText = closestValue.toString();
    } else {
      tooltipRef.current.style.display = 'none';
    }

    return position;
  }, [ data, tooltipRef, trialIdRef, trialIds, xValues ]);

  useEffect(() => {
    if (!chartRef.current) return;

    const now = Date.now();
    const options = uPlot.assign({}, UPLOT_OPTIONS, {
      cursor: {
        bind: { mouseleave: handleMouseLeave },
        move: handleCursorMove,
      },
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

    const chart = new uPlot(options, [ xValues, ...data ], chartRef.current);
    console.log('render time', (Date.now() - now) / 1000);

    return () => chart.destroy();
    /* eslint-disable-next-line react-hooks/exhaustive-deps */
  }, [ data, xValues ]);

  return (
    <div className={css.base}>
      <div ref={chartRef} />
      <div className={css.tooltip} ref={tooltipRef}>
        <div className={css.point} />
        <div className={css.box}>
          <div className={css.row}>
            <div>Trial Id:</div>
            <div ref={trialIdRef} />
          </div>
          <div className={css.tooltipRow}>
            <div>Batches:</div>
            <div ref={batchesRef} />
          </div>
          <div className={css.tooltipRow}>
            <div>Metric Value:</div>
            <div ref={metricValueRef} />
          </div>
        </div>
      </div>
    </div>
  );
};

export default LearningCurveChart;
