import React, { useMemo, useRef } from 'react';
import uPlot from 'uplot';

import QuadTree, { pointWithin } from 'components/UPlot/quadtree';
import UPlotChart, { Options } from 'components/UPlotChart';

import { FacetedData, getSize, getSizeMinMax, makeDrawPoints, UPlotData } from './utils';

interface Props {
  data?: FacetedData;
  options?: Partial<Options>;
  style?: React.CSSProperties;
}

const UPlotScatterPlot: React.FC<Props> = ({ data, options, style }: Props) => {
  const quadtree = useRef<QuadTree>();
  const hRect = useRef<QuadTree | null>();

  const drawPoints = useMemo(() => {
    return makeDrawPoints({
      disp: {
        size: {
          unit: 3, // raw CSS pixels
          //	discr: true,
          values: (u, seriesIdx, idx0, idx1) => {
            // TODO: only run once per setData() call
            const [ minValue, maxValue ] = getSizeMinMax(u);
            const seriesData = (u?.data[seriesIdx][2] || []) as unknown as UPlotData[];
            return seriesData.map(v => getSize(v, minValue, maxValue)) || [];
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
    return {
      axes: [
        { label: 'GDP' },
        { label: 'Income 1' },
        {
          grid: { show: false },
          label: 'Income 2',
          scale: 'y2',
          side: 1,
          stroke: 'red',
        },
      ],
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
      height: 400,
      hooks: {
        drawClear: [
          u => {
            quadtree.current = quadtree.current || new QuadTree(0, 0, u.bbox.width, u.bbox.height);
            quadtree.current.clear();

            // force-clear the path cache to cause drawBars() to rebuild new quadtree
            u.series.forEach((s, i) => {
              if (i > 0) (s as unknown as { _paths: uPlot.Series.Paths | null })._paths = null;
            });
          },
        ],
      },
      mode: 2,
      scales: {
        x: {
          range: (u, min, max) => [ min, max ],
          time: false,
        },
        y: { range: (u, min, max) => [ min, max ] },
        y2: {
          dir: 1,
          ori: 1,
          range: (u, min, max) => [ min, max ],
        },
      },
      series: [
        null,
        {
          facets: [
            { auto: true, scale: 'x' },
            { auto: true, scale: 'y2' },
          ],
          fill: 'rgba(255,0,0,0.3)',
          label: 'Region A',
          paths: drawPoints,
          stroke: 'red',
        },
        {
          facets: [
            { auto: true, scale: 'x' },
            { auto: true, scale: 'y' },
          ],
          fill: 'rgba(0,255,0,0.3)',
          label: 'Region B',
          paths: drawPoints,
          stroke: 'green',
        },
        {
          facets: [
            { auto: true, scale: 'x' },
            { auto: true, scale: 'y' },
          ],
          fill: 'rgba(0,0,255,0.3)',
          label: 'Region C',
          paths: drawPoints,
          stroke: 'blue',
        },
        {
          facets: [
            { auto: true, scale: 'x' },
            { auto: true, scale: 'y' },
          ],
          fill: 'rgba(255,128,0,0.3)',
          label: 'Region E',
          paths: drawPoints,
          stroke: 'orange',
        },
      ],
      title: 'Bubble Plot',
    } as Partial<Options>;
  }, [ drawPoints ]);

  return (
    <UPlotChart data={data} options={chartOptions} />
  );
};

export default UPlotScatterPlot;
