import uPlot from 'uplot';

import { Range } from 'types';

import { UPlotAxisSplits, UPlotData } from './types';

const DEFAULT_LOG_BASE = 10;
const MIN_DIAMETER = 6;
const MAX_DIAMETER = 60;
const MIN_AREA = Math.PI * (MIN_DIAMETER / 2) ** 2;
const MAX_AREA = Math.PI * (MAX_DIAMETER / 2) ** 2;

// quadratic scaling (px area)
export const getSize = (
  value: number | null | undefined,
  minValue: number,
  maxValue: number,
): number => {
  if (value == null || minValue === maxValue || minValue == null) return 0;
  const pct = (value - minValue) / (maxValue - minValue);
  const area = (MAX_AREA - MIN_AREA) * pct + MIN_AREA;
  return Math.sqrt(area / Math.PI) * 2;
};

export const getSizeMinMax = (u: uPlot): [ number, number ] => {
  let minValue = Infinity;
  let maxValue = -Infinity;

  for (let i = 1; i < u.series.length; i++) {
    const sizeData = u.data[i][2] as unknown as number[];

    for (let j = 0; j < sizeData.length; j++) {
      minValue = Math.min(minValue, sizeData[j]);
      maxValue = Math.max(maxValue, sizeData[j]);
    }
  }

  return [ minValue, maxValue ];
};

export const range = (
  u: uPlot,
  min: UPlotData,
  max: UPlotData,
  scaleKey: string,
): Range<number> => {
  // Return a standard range if there is not any valid data.
  if (min == null || max == null) return [ 0, 100 ];

  // When there is only one distinct value in the dataset.
  if (min === max) {
    const axisIndex = u.axes.findIndex(axis => axis.scale === scaleKey);
    const axis = u.axes[axisIndex] as uPlot.Axis | undefined;
    const getSplits = axis?.splits as UPlotAxisSplits | undefined;

    if (/categorical/i.test(scaleKey)) {
      // Using splits for categorical axis.
      const splits = getSplits?.(u, axisIndex, min, max) || [];
      if (splits.length > 1) return [ splits.first(), splits.last() ];
    } else if (/log/i.test(scaleKey)) {
      const logBase = u.scales[scaleKey].log || DEFAULT_LOG_BASE;
      const nearestBase = Math.log(max) / Math.log(logBase);
      return [ logBase ** (nearestBase - 1), logBase ** (nearestBase + 1) ];
    } else {
      const delta = Math.abs(max) || 100;
      return [ min - delta, max + delta ];
    }
  }

  return [ min as number, max as number ];
};

export const offsetRange = (offsetPercent = 0.1) => {
  return (u: uPlot, min: UPlotData, max: UPlotData, scaleKey: string): Range<number> => {
    const [ minValue, maxValue ] = range(u, min, max, scaleKey);

    // Offset log scale based on the exponents.
    if (/log/i.test(scaleKey)) {
      const logBase = u.scales[scaleKey].log || DEFAULT_LOG_BASE;
      const minLog = Math.log(minValue) / Math.log(logBase);
      const maxLog = Math.log(maxValue) / Math.log(logBase);
      const offset = (maxLog - minLog) * offsetPercent;
      return [ logBase ** (minLog - offset), logBase ** (maxLog + offset) ];
    }

    const offset = (maxValue - minValue) * offsetPercent;
    return [ minValue - offset, maxValue + offset ];
  };
};

export const makeDrawPoints = (
  options: uPlot.Series.BarsPathBuilderOpts,
): uPlot.Series.PathBuilder => {
  const { disp, each } = options;

  return (u: uPlot, seriesIdx: number, idx0: number, idx1: number) => {
    uPlot.orient(u, seriesIdx, (
      series,
      dataX,
      dataY,
      scaleX,
      scaleY,
      valToPosX,
      valToPosY,
      xOff,
      yOff,
      xDim,
      yDim,
      // moveTo,
      // lineTo,
      // rect,
      // arc,
    ) => {
      if (!series?.fill || !series?.stroke || !disp?.size || !scaleX?.key || !scaleY?.key) return;

      const data = u.data[seriesIdx];
      const strokeWidth = 1;

      u.ctx.save();

      u.ctx.rect(u.bbox.left, u.bbox.top, u.bbox.width, u.bbox.height);
      u.ctx.clip();

      u.ctx.fillStyle = typeof series.fill === 'function'
        ? series.fill(u, seriesIdx) : series.fill;
      u.ctx.strokeStyle = typeof series.stroke === 'function'
        ? series.stroke(u, seriesIdx) : series.stroke;
      u.ctx.lineWidth = strokeWidth;

      // compute bubble dims
      const sizes = disp.size.values(u, seriesIdx, idx0, idx1) as unknown as number[];

      // todo: this depends on direction & orientation
      // todo: calc once per redraw, not per path
      const filtLft = u.posToVal(-MAX_DIAMETER / 2, scaleX.key);
      const filtRgt = u.posToVal(u.bbox.width / devicePixelRatio + MAX_DIAMETER / 2, scaleX.key);
      const filtBtm = u.posToVal(u.bbox.height / devicePixelRatio + MAX_DIAMETER / 2, scaleY.key);
      const filtTop = u.posToVal(-MAX_DIAMETER / 2, scaleY.key);

      const xData = data[0] as unknown as number[];
      const yData = data[1] as unknown as UPlotData[];

      for (let i = 0; i < xData.length; i++) {
        const xVal = xData[i];
        const yVal = yData[i] ?? 0;
        const size = sizes[i] * devicePixelRatio;

        if (xVal >= filtLft && xVal <= filtRgt && yVal >= filtBtm && yVal <= filtTop) {
          const cx = valToPosX(xVal, scaleX, xDim, xOff);
          const cy = valToPosY(yVal, scaleY, yDim, yOff);

          u.ctx.moveTo(cx + size / 2, cy);
          u.ctx.beginPath();
          u.ctx.arc(cx, cy, size / 2, 0, 2 * Math.PI);
          u.ctx.fill();
          u.ctx.stroke();

          each?.(
            u,
            seriesIdx,
            i,
            cx - size / 2 - strokeWidth / 2,
            cy - size / 2 - strokeWidth / 2,
            size + strokeWidth,
            size + strokeWidth,
          );
        }
      }

      u.ctx.restore();
    });

    return null;
  };
};
