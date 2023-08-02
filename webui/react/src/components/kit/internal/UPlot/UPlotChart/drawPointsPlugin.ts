import uPlot, { Plugin } from 'uplot';

import { isNumber } from 'components/kit/internal/functions';
import { CheckpointsDict } from 'components/kit/internal/types';

const NUM_POINTS = 4;
const OUTER_RADIUS = 5;
const INNER_RADIUS = 3;
const outerRadius = OUTER_RADIUS * devicePixelRatio;
const innerRadius = INNER_RADIUS * devicePixelRatio;

export const drawPointsPlugin = (checkpointsDict: CheckpointsDict): Plugin => {
  function drawCheckpoint(ctx: CanvasRenderingContext2D, cx: number, cy: number) {
    let rot = (Math.PI / 2) * 3;
    let x = cx;
    let y = cy;
    const step = Math.PI / NUM_POINTS;

    ctx.beginPath();
    ctx.moveTo(cx, cy - outerRadius);

    for (let i = 0; i < NUM_POINTS; i++) {
      x = cx + Math.cos(rot) * outerRadius;
      y = cy + Math.sin(rot) * outerRadius;
      ctx.lineTo(x, y);
      rot += step;

      x = cx + Math.cos(rot) * innerRadius;
      y = cy + Math.sin(rot) * innerRadius;
      ctx.lineTo(x, y);
      rot += step;
    }

    ctx.lineTo(cx, cy - outerRadius);
    ctx.closePath();
  }

  // function drawPoints(u: uPlot, i, i0, i1) {
  function drawPoints(u: uPlot, seriesIdx: number, idx0: number, idx1: number) {
    const { ctx } = u;
    const { scale } = u.series[seriesIdx];

    ctx.save();

    let j = idx0;
    let foundCheckpoint = false;

    while (j <= idx1) {
      const xVal = u.data[0][j];
      const yVal = u.data[seriesIdx][j];

      if (scale && isNumber(yVal) && checkpointsDict[Math.floor(xVal)]) {
        foundCheckpoint = true;
        const cx = Math.round(u.valToPos(xVal, 'x', true));
        const cy = Math.round(u.valToPos(yVal, scale, true));
        drawCheckpoint(ctx, cx, cy);
        const blankFill = getComputedStyle(ctx.canvas).getPropertyValue('--theme-stage');
        ctx.fillStyle = blankFill;
        ctx.fill();
        ctx.stroke();
      }

      j++;
    }

    ctx.restore();
    if (!foundCheckpoint && u.data[seriesIdx].length === 1) {
      return true;
    }
  }

  return {
    hooks: {},
    opts: (u, opts) => {
      opts.series.forEach((s, i) => {
        if (i > 0) {
          uPlot.assign(s, {
            points: {
              show: drawPoints,
            },
          });
        }
      });
    },
  };
};
