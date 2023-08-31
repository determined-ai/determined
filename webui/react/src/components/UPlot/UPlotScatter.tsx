import React, { useMemo, useRef } from 'react';
import uPlot from 'uplot';

import { FacetedData, UPlotData } from 'components/UPlot/types';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';
import QuadTree, { pointWithin } from 'components/UPlot/UPlotScatter/quadtree';
import {
  FILL_INDEX,
  getColorFn,
  getMinMax,
  getSize,
  makeDrawPoints,
  offsetRange,
  SIZE_INDEX,
  STROKE_INDEX,
} from 'components/UPlot/UPlotScatter/UPlotScatter.utils';
import useScatterPointTooltipPlugin from 'components/UPlot/UPlotScatter/useScatterPointTooltipPlugin';
import css from 'components/UPlot/UPlotScatter.module.scss';
import { Range, Scale } from 'types';

interface Props {
  colorScaleDistribution?: Scale;
  data?: FacetedData;
  options?: Partial<Options>;
  tooltipLabels?: (string | null)[];
}

const DEFAULT_FILL_COLOR = 'rgba(0, 155, 222, 0.3)';
const DEFAULT_HOVER_COLOR = 'rgba(0, 155, 222, 1.0)';
const DEFAULT_STROKE_COLOR = 'rgba(0, 155, 222, 1.0)';

const UPlotScatter: React.FC<Props> = ({
  data,
  colorScaleDistribution,
  options = {},
  tooltipLabels,
}: Props) => {
  const quadtree = useRef<QuadTree>();
  const hRect = useRef<QuadTree | null>();
  const ranges = useRef<Range<number>[]>([]);

  const tooltipPlugin = useScatterPointTooltipPlugin({ labels: tooltipLabels });

  const drawPoints = useMemo(() => {
    return makeDrawPoints({
      disp: {
        fill: {
          unit: 3,
          values: (u, seriesIndex) => {
            const getColor = getColorFn(u.series[seriesIndex].fill, DEFAULT_FILL_COLOR);
            const yData = (u?.data[seriesIndex][1] || []) as unknown as UPlotData[];
            const fillData = u?.data[seriesIndex][FILL_INDEX];
            if (fillData === null) return yData.map(() => getColor(0, 0, 1));

            const [minValue, maxValue] = ranges.current[FILL_INDEX];
            const seriesData = (fillData || []) as unknown as UPlotData[];
            return (
              seriesData.map((value) =>
                getColor(value, minValue, maxValue, colorScaleDistribution),
              ) || []
            );
          },
        },
        size: {
          unit: 3,
          values: (u, seriesIndex) => {
            const yData = (u?.data[seriesIndex][1] || []) as unknown as UPlotData[];
            const sizeData = u?.data[seriesIndex][SIZE_INDEX];
            if (sizeData === null) return yData.map(() => getSize(0, 0, 1));

            const [minValue, maxValue] = ranges.current[SIZE_INDEX];
            const seriesData = (sizeData || []) as unknown as UPlotData[];
            return seriesData.map((value) => getSize(value, minValue, maxValue)) || [];
          },
        },
        stroke: {
          unit: 3,
          values: (u, seriesIndex) => {
            const getColor = getColorFn(u.series[seriesIndex].stroke, DEFAULT_STROKE_COLOR);
            const yData = (u?.data[seriesIndex][1] || []) as unknown as UPlotData[];
            const strokeData = u?.data[seriesIndex][STROKE_INDEX];
            if (strokeData === null) return yData.map(() => getColor(0, 0, 1));

            const [minValue, maxValue] = ranges.current[STROKE_INDEX];
            const seriesData = (strokeData || []) as unknown as UPlotData[];
            return (
              seriesData.map((value) =>
                getColor(value, minValue, maxValue, colorScaleDistribution),
              ) || []
            );
          },
        },
      },
      each: (u, seriesIndex, dataIndex, left, top, width, height) => {
        /**
         * We get back raw canvas coords (included axes & padding).
         * Translate to the plotting area origin.
         */
        left -= u.bbox.left;
        top -= u.bbox.top;
        quadtree.current?.add(
          new QuadTree(left, top, width, height, undefined, seriesIndex, dataIndex),
        );
      },
    });
  }, [colorScaleDistribution]);

  const chartOptions = useMemo(() => {
    const seriesOptions = options.series?.[1] || {};
    return uPlot.assign(
      {
        cursor: {
          dataIdx: (u, seriesIndex) => {
            if (seriesIndex === 1) {
              let dist = Infinity;
              const cx = (u.cursor.left || 0) * devicePixelRatio;
              const cy = (u.cursor.top || 0) * devicePixelRatio;

              hRect.current = null;
              quadtree.current?.get(cx, cy, 1, 1, (o) => {
                if (pointWithin(cx, cy, o.x, o.y, o.x + o.w, o.y + o.h)) {
                  const ocx = o.x + o.w / 2;
                  const ocy = o.y + o.h / 2;

                  const dx = ocx - cx;
                  const dy = ocy - cy;

                  const d = Math.sqrt(dx ** 2 + dy ** 2);

                  // Test against radius for actual hover.
                  if (d <= o.w / 2) {
                    // Only hover bbox with closest distance.
                    if (d <= dist) {
                      dist = d;
                      hRect.current = o;
                    }
                  }
                }
              });
            }

            return seriesIndex === hRect.current?.seriesIndex ? hRect.current.dataIndex : null;
          },
          points: {
            fill: (u, seriesIndex) => {
              const dataIndex = u?.cursor?.dataIdx?.(u, seriesIndex, 0, 0);
              if (dataIndex == null) return DEFAULT_HOVER_COLOR;

              const getColor = getColorFn(u.series[1].stroke, DEFAULT_HOVER_COLOR);
              const fillData = u?.data?.[seriesIndex]?.[FILL_INDEX] as unknown as UPlotData[];
              const value = (fillData || [])[dataIndex];
              if (value == null) return getColor(0, 0, 1);

              const [minValue, maxValue] = ranges.current[FILL_INDEX];
              return getColor(value, minValue, maxValue, colorScaleDistribution);
            },
            size: (u, seriesIndex) => {
              return seriesIndex === hRect.current?.seriesIndex
                ? hRect.current.w / devicePixelRatio
                : 0;
            },
          },
        },
        height: 350,
        hooks: {
          drawClear: [
            (u) => {
              quadtree.current =
                quadtree.current || new QuadTree(0, 0, u.bbox.width, u.bbox.height);
              quadtree.current.clear();

              // force-clear the path cache to cause drawBars() to rebuild new quadtree
              u.series.forEach((s, i) => {
                if (i > 0) (s as unknown as { _paths: uPlot.Series.Paths | null })._paths = null;
              });
            },
          ],
          setData: [
            (u) => {
              // Calculate the min and max of each data properties such as size, fill and stroke.
              (u.data[1] || []).forEach((data, index) => {
                if (data != null) ranges.current[index] = getMinMax(u, index);
              });
            },
          ],
        },
        legend: { show: false },
        mode: 2,
        padding: [0, 8, 0, 8],
        plugins: [tooltipPlugin],
        scales: {
          x: { range: offsetRange(), time: false },
          xCategorical: { range: offsetRange(), time: false },
          xLog: { distr: 3, log: 10, range: offsetRange(), time: false },
          y: { range: offsetRange() },
          yLog: { distr: 3, range: offsetRange() },
        },
      } as Partial<Options>,
      options,
      // Override paths drawing option to support faceted data drawing.
      {
        series: [
          null,
          {
            ...seriesOptions,
            facets: [
              { auto: true, scale: options.axes?.[0].scale || 'x' },
              { auto: true, scale: options.axes?.[1].scale || 'y' },
            ],
            fill: seriesOptions.fill,
            paths: drawPoints,
            stroke: seriesOptions.stroke,
          },
        ],
      } as Partial<Options>,
    );
  }, [colorScaleDistribution, drawPoints, options, tooltipPlugin]);

  return (
    <div className={css.base}>
      <UPlotChart data={data} options={chartOptions} />
    </div>
  );
};

export default UPlotScatter;
