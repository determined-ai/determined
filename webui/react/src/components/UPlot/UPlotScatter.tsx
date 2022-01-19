import React, { useMemo, useRef } from 'react';
import uPlot from 'uplot';

import QuadTree, { pointWithin } from 'components/UPlot/quadtree';
import UPlotChart, { Options } from 'components/UPlot/UPlotChart';

import { FacetedData, UPlotData } from './types';
import css from './UPlotScatter.module.scss';
import {
  FILL_INDEX, getColorFn, getSize, getSizeMinMax, makeDrawPoints, offsetRange, SIZE_INDEX,
} from './UPlotScatter.utils';

interface Props {
  data?: FacetedData;
  options?: Partial<Options>;
}

const UPlotScatter: React.FC<Props> = ({ data, options = {} }: Props) => {
  const quadtree = useRef<QuadTree>();
  const hRect = useRef<QuadTree | null>();

  const drawPoints = useMemo(() => {
    return makeDrawPoints({
      disp: {
        fill: {
          unit: 3,
          values: (u, seriesIndex) => {
            const getColor = getColorFn(u.series[1].fill);
            const yData = (u?.data[seriesIndex][1] || []) as unknown as UPlotData[];
            const fillData = u?.data[seriesIndex][FILL_INDEX];
            if (fillData === null) return yData.map(() => getColor(0, 0, 1));

            const [ minValue, maxValue ] = getSizeMinMax(u, FILL_INDEX);
            const seriesData = (fillData || []) as unknown as UPlotData[];
            return seriesData.map(value => getColor(value, minValue, maxValue)) || [];
          },
        },
        size: {
          unit: 3,
          values: (u, seriesIndex) => {
            // TODO: only run once per setData() call
            const yData = (u?.data[seriesIndex][1] || []) as unknown as UPlotData[];
            const sizeData = u?.data[seriesIndex][SIZE_INDEX];
            if (sizeData === null) return yData.map(() => getSize(0, 0, 1));

            const [ minValue, maxValue ] = getSizeMinMax(u, SIZE_INDEX);
            const seriesData = (sizeData || []) as unknown as UPlotData[];
            return seriesData.map(value => getSize(value, minValue, maxValue)) || [];
          },
        },
      },
      each: (u, seriesIdx, dataIdx, lft, top, wid, hgt) => {
        // we get back raw canvas coords (included axes & padding).
        // translate to the plotting area origin
        lft -= u.bbox.left;
        top -= u.bbox.top;
        quadtree.current?.add(new QuadTree(
          lft,
          top,
          wid,
          hgt,
          undefined,
          seriesIdx,
          dataIdx,
        ));
      },
    });
  }, []);

  const chartOptions = useMemo(() => {
    return uPlot.assign(
      {
        cursor: {
          dataIdx: (u, seriesIndex) => {
            if (seriesIndex === 1) {
              let dist = Infinity;
              const cx = (u.cursor.left || 0) * devicePixelRatio;
              const cy = (u.cursor.top || 0) * devicePixelRatio;

              hRect.current = null;
              quadtree.current?.get(cx, cy, 1, 1, o => {
                if (pointWithin(cx, cy, o.x, o.y, o.x + o.w, o.y + o.h)) {
                  const ocx = o.x + o.w / 2;
                  const ocy = o.y + o.h / 2;

                  const dx = ocx - cx;
                  const dy = ocy - cy;

                  const d = Math.sqrt(dx ** 2 + dy ** 2);

                  // test against radius for actual hover
                  if (d <= o.w / 2) {
                  // only hover bbox with closest distance
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
            size: (u, seriesIndex) => {
              return seriesIndex === hRect.current?.seriesIndex
                ? hRect.current.w / devicePixelRatio : 0;
            },
          },
        },
        height: 350,
        hooks: {
          drawClear: [
            u => {
              quadtree.current = quadtree.current
                || new QuadTree(0, 0, u.bbox.width, u.bbox.height);
              quadtree.current.clear();

              // force-clear the path cache to cause drawBars() to rebuild new quadtree
              u.series.forEach((s, i) => {
                if (i > 0) (s as unknown as { _paths: uPlot.Series.Paths | null })._paths = null;
              });
            },
          ],
        },
        legend: { show: false },
        mode: 2,
        padding: [ 0, 8, 0, 8 ],
        scales: {
          x: { range: offsetRange(), time: false },
          xCategorical: { range: offsetRange(), time: false },
          xLog: { distr: 3, log: 10, range: offsetRange(), time: false },
          y: { range: offsetRange() },
          yLog: { distr: 3, range: offsetRange() },
        },
        series: [
          null,
          {
            facets: [
              { auto: true, scale: options.axes?.[0].scale || 'x' },
              { auto: true, scale: options.axes?.[1].scale || 'y' },
            ],
            fill: 'rgba(255,0,0,0.3) rgba(0,0,255,0.3)',
            paths: drawPoints,
            stroke: 'rgba(255,0,0,1)',
          },
        ],
      } as Partial<Options>,
      options || {},
    );
  }, [ drawPoints, options ]);

  return (
    <div className={css.base}>
      <UPlotChart data={data} options={chartOptions} />
    </div>
  );
};

export default UPlotScatter;
