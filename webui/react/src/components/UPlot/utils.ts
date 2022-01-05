import uPlot from 'uplot';

export type UPlotData = number | null | undefined;
export type FacetedData = [
  xValues: null,
  ...yValues: (UPlotData[] | string[])[],
];

const MAX_DIAMETER = 60;
const MAX_AREA = Math.PI * (MAX_DIAMETER / 2) ** 2;

export const randomNumber = (min: number, max: number): number => {
  return Math.floor(Math.random() * (max - min + 1)) + min;
};

export const filledArray = (
  len: number,
  val: number | ((i: number) => number | number[][] | string),
): number | number[][] | string => {
  const arr = new Array(len);

  if (typeof val == 'function') {
    for (let i = 0; i < len; ++i) arr[i] = val(i);
  } else {
    for (let i = 0; i < len; ++i) arr[i] = val;
  }

  return arr;
};

// quadratic scaling (px area)
export const getSize = (
  value: number | null | undefined,
  minValue: number,
  maxValue: number,
): number => {
  if (value == null) return 0;
  const pct = value / maxValue;
  const area = MAX_AREA * pct;
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
      console.log(
        'makeDrawPoints',
        u.ctx.fillStyle,
        u.ctx.strokeStyle,
        (series.fill as ((u: uPlot, sidx: number) => string))(u, seriesIdx),
      );

      const deg360 = 2 * Math.PI;

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
          u.ctx.arc(cx, cy, size / 2, 0, deg360);
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
