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
  onPointFocus?: (point: Point|undefined) => void,
  yScale: string, // y scale to use
}

export const closestPointPlugin = (
  { distInPx = 30, onPointFocus, yScale }: Props,
): Plugin => {
  let distValX: number; // distInPx transformed to X value
  let distValY: number; // distInPx transformed to Y value
  let focusedPoint: Point|undefined; // focused data point
  let pointEl: HTMLDivElement;

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
    const yVal = point && uPlot.data[point.seriesIdx][point.idx];

    // point
    if (point && series && yVal != null) {

      pointEl.style.backgroundColor = typeof series.stroke === 'function'
        ? series.stroke(uPlot, point.seriesIdx) as string : 'rgba(0, 155, 222, 1)';
      pointEl.style.display = 'block';
      pointEl.style.left = uPlot.valToPos(uPlot.data[0][point.idx], 'x') + 'px';
      pointEl.style.top = uPlot.valToPos(yVal, yScale) + 'px';
    } else {
      pointEl.style.display = 'none';
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

  // let displayedIdx: number|null = null;
  // let tooltipEl: HTMLDivElement|null = null;
  //
  // const _buildTooltipHtml = (uPlot: uPlot, idx: number): string => {
  //   let html = '';
  //
  //   let header: ChartTooltip = null;
  //   if (typeof getXTooltipHeader === 'function') {
  //     header = getXTooltipHeader(idx);
  //   }
  //   let yLabels: ChartTooltip[] = [];
  //   if (typeof getXTooltipYLabels === 'function') {
  //     yLabels = getXTooltipYLabels(idx);
  //   }
  //
  //   const xSerie = uPlot.series[0];
  //   const xValue = (typeof xSerie.value === 'function' ?
  //     xSerie.value(uPlot, uPlot.data[0][idx], 0, idx) : uPlot.data[0][idx]);
  //   html += `<div class="${css.valueX}">`
  //     + (header ? header + '<br />' : '')
  //     + `${xSerie.label}: ${xValue}`
  //     + '</div>';
  //
  //   uPlot.series.forEach((serie, i) => {
  //     if (serie.scale === 'x' || !serie.show) return;
  //
  //     const label = yLabels[i - 1] || null;
  //     const valueRaw = uPlot.data[i][idx];
  //
  //     const cssClass = valueRaw ? css.valueY : css.valueYEmpty;
  //     html += `<div class="${cssClass}">`
  //       + `<span class="${css.color}" style="background-color: ${glasbeyColor(i - 1)}"></span>`
  //       + (label ? label + '<br />' : '')
  //       + `${serie.label}: ${valueRaw || 'N/A'}`
  //       + '</div>';
  //   });
  //
  //   return html;
  // };
  //
  // const _getTooltipLeftPx = (uPlot: uPlot, idx: number): number => {
  //   const idxLeft = uPlot.valToPos(uPlot.data[0][idx], 'x');
  //   if (!tooltipEl) return idxLeft;
  //
  //   const chartWidth = uPlot.root.querySelector('.u-over')?.getBoundingClientRect().width;
  //   const tooltipWidth = tooltipEl.getBoundingClientRect().width;
  //
  //   // right
  //   if (chartWidth && idxLeft + tooltipWidth >= chartWidth) {
  //     return (idxLeft - tooltipWidth);
  //   }
  //
  //   // left
  //   return idxLeft;
  // };
  //
  // const _updateTooltipVerticalPosition = (uPlot: uPlot, cursorTop: number) => {
  //   if (!tooltipEl) return;
  //
  //   const chartHeight = uPlot.root.querySelector('.u-over')?.getBoundingClientRect().height;
  //
  //   const vPos = (chartHeight && cursorTop > (chartHeight/2)) ? 'top' : 'bottom';
  //
  //   tooltipEl.style.bottom = vPos === 'bottom' ? '0px' : 'auto';
  //   tooltipEl.style.top = vPos === 'top' ? '0px' : 'auto';
  // };
  //
  // const showIdx = (uPlot: uPlot, idx: number) => {
  //   if (!tooltipEl || !barEl) return;
  //   displayedIdx = idx;
  //
  //   const idxLeft = uPlot.valToPos(uPlot.data[0][idx], 'x');
  //
  //   barEl.style.display = 'block';
  //   barEl.style.left = idxLeft + 'px';
  //
  //   tooltipEl.innerHTML = _buildTooltipHtml(uPlot, idx);
  //   tooltipEl.style.display = 'block';
  //   tooltipEl.style.left = _getTooltipLeftPx(uPlot, idx) + 'px';
  // };
  //
  // const hide = () => {
  //   if (!tooltipEl || !barEl) return;
  //   displayedIdx = null;
  //
  //   barEl.style.display = 'none';
  //   tooltipEl.style.display = 'none';
  // };

  return {
    hooks: {
      ready: (uPlot: uPlot) => {
        pointEl = document.createElement('div');
        pointEl.className = css.point;
        uPlot.root.querySelector('.u-over')?.appendChild(pointEl);

        // barEl = document.createElement('div');
        // barEl.className = css.bar;
        // uPlot.root.querySelector('.u-over')?.appendChild(barEl);
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
