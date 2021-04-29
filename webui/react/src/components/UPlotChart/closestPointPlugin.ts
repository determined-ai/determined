import { throttle } from 'throttle-debounce';
import uPlot, { Options, Plugin } from 'uplot';

import { findInsertionIndex } from 'utils/array';
import { distance } from 'utils/chart';
import { isEqual } from 'utils/data';

import css from './closestPointPlugin.module.scss';

interface Point {
  idx: number;
  seriesIdx: number;
}

interface Props {
  distInPx?: number, // max cursor distance from data point to focus it (in pixel)
  getPointTooltipHTML?: (xVal: number, yVal: number, point: Point) => string,
  onPointClick?: (e: MouseEvent, point: Point) => void,
  onPointFocus?: (point: Point|undefined) => void,
  pointSizeInPx?: number,
  yScale: string, // y scale to use
}

export const closestPointPlugin = ({
  distInPx = 30,
  getPointTooltipHTML,
  onPointClick,
  onPointFocus,
  pointSizeInPx = 7,
  yScale,
}: Props): Plugin => {
  let distValX: number; // distInPx transformed to X value
  let distValY: number; // distInPx transformed to Y value
  let focusedPoint: Point|undefined; // focused data point
  let pointEl: HTMLDivElement;
  let tooltipEl: HTMLDivElement;

  const findClosestPoint =
    (uPlot: uPlot, cursorLeft: number, cursorTop: number): Point|undefined => {
      let closestDistance: number = Number.MAX_VALUE;
      let closestPoint: Point|undefined;

      // find idx range
      // note: assuming X data to be sorted, uPlot behaves odd if that's false
      const cursorValX = uPlot.posToVal(cursorLeft, 'x');
      const idxMax = findInsertionIndex(uPlot.data[0], cursorValX + distValX) - 1;
      const idxMin = findInsertionIndex(uPlot.data[0], cursorValX - distValX);

      // find y value range
      const cursorValY = uPlot.posToVal(cursorTop, yScale);
      const yValMax = cursorValY + distValY;
      const yValMin = cursorValY - distValY;

      // cycle on each data point in the idx range found
      for (let idx = idxMin; idx <= idxMax; idx++) {
        const posX = uPlot.valToPos(uPlot.data[0][idx], 'x');

        for (let seriesIdx = 1; seriesIdx < uPlot.data.length; seriesIdx++) {
          const yVal = uPlot.data[seriesIdx][idx];

          // value is inside Y range
          if (yVal && yVal >= yValMin && yVal <= yValMax) {
            const posY = uPlot.valToPos(yVal, yScale);

            const yValDistance = distance(posX, posY, cursorLeft, cursorTop);
            if (yValDistance < closestDistance) {
              closestDistance = yValDistance;
              closestPoint = { idx, seriesIdx };
            }
          }
        }
      }

      return closestPoint;
    };

  const focusPoint = (uPlot: uPlot, point: Point|undefined) => {
    focusedPoint = point;

    if (typeof onPointFocus === 'function') {
      onPointFocus(focusedPoint);
    }

    const series = point && uPlot.series[point.seriesIdx];
    const xVal = point && uPlot.data[0][point.idx];
    const yVal = point && uPlot.data[point.seriesIdx][point.idx];

    const xPos = point && uPlot.valToPos(uPlot.data[0][point.idx], 'x');
    const yPos = yVal && uPlot.valToPos(yVal, yScale);

    if (!point || !series || xVal == null || yVal == null || xPos == null || yPos == null) {
      pointEl.style.display = 'none';
      tooltipEl.style.display = 'none';
      return;
    }

    // point
    if (pointSizeInPx > 0) {
      pointEl.style.backgroundColor = typeof series.stroke === 'function'
        ? series.stroke(uPlot, point.seriesIdx) as string : 'rgba(0, 155, 222, 1)';
      pointEl.style.display = 'block';
      pointEl.style.height = pointSizeInPx + 'px';
      pointEl.style.left = xPos + 'px';
      pointEl.style.top = yPos + 'px';
      pointEl.style.width = pointSizeInPx + 'px';
    }

    // tooltip
    const tooltipHtml = typeof getPointTooltipHTML === 'function'
      && getPointTooltipHTML(xVal, yVal, point);
    if (tooltipHtml) {
      const classes = [ css.tooltip ];
      if (xPos > uPlot.bbox.width / 2 / window.devicePixelRatio) classes.push(css.left);
      if (yPos > uPlot.bbox.height / 2 / window.devicePixelRatio) classes.push(css.top);

      tooltipEl.className = classes.join(' ');
      tooltipEl.innerHTML = `<div class="${css.box}">${tooltipHtml}</div>`;
      tooltipEl.style.display = 'block';
      tooltipEl.style.left = xPos + 'px';
      tooltipEl.style.top = yPos + 'px';
    } else {
      tooltipEl.style.display = 'none';
    }
  };

  const handleCursorMove = throttle(100, (uPlot: uPlot) => {
    const { left, idx, top } = uPlot.cursor;

    if (!left || left < 0 || !top || top < 0 || idx == null) {
      if (focusedPoint) focusPoint(uPlot, undefined);
      return;
    }

    const point = findClosestPoint(uPlot, left, top);
    if (!point) {
      if (focusedPoint) focusPoint(uPlot, undefined);
      return;
    }

    if (!isEqual(point, focusedPoint)) {
      focusPoint(uPlot, point);
    }
  });

  return {
    hooks: {
      ready: (uPlot: uPlot) => {
        const over = uPlot.root.querySelector('.u-over');
        if (!over) return;

        // point div
        pointEl = document.createElement('div');
        pointEl.className = css.point;
        over.appendChild(pointEl);

        // point div
        tooltipEl = document.createElement('div');
        over.appendChild(tooltipEl);

        // click handler
        if (typeof onPointClick === 'function') {
          let mousedownX: number;
          let mousedownY: number;
          over.addEventListener('mousedown', e => {
            mousedownX = (e as MouseEvent).clientX;
            mousedownY = (e as MouseEvent).clientY;
          });
          over.addEventListener('mouseup', e => {
            if (
              (e as MouseEvent).clientX !== mousedownX
              || (e as MouseEvent).clientY !== mousedownY
              || !focusedPoint
            ) return;

            onPointClick(e as MouseEvent, focusedPoint);
          });
        }
      },
      setCursor: (uPlot: uPlot) => handleCursorMove(uPlot),
      setScale: (uPlot: uPlot) => {
        distValX = uPlot.posToVal(distInPx, 'x') - uPlot.posToVal(0, 'x');
        distValY = uPlot.posToVal(0, yScale) - uPlot.posToVal(distInPx, yScale);
      },
    },
    opts: (self, opts) => {
      return uPlot.assign({}, opts, { cursor: { points: { show: false } } }) as Options;
    },
  };
};
