import Hermes from 'hermes-parallel-coordinates';
import React, { useEffect, useRef } from 'react';

import css from './ParallelCoordinates.module.scss';

interface Props {
  config?: Hermes.RecursivePartial<Hermes.Config>;
  data: Hermes.Data;
  dimensions: Hermes.Dimension[];
  height?: number;
}

const ParallelCoordinates: React.FC<Props> = ({
  config,
  data,
  dimensions,
  height = 450,
}: Props) => {
  const chartRef = useRef<Hermes>();
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    chartRef.current = new Hermes(containerRef.current);

    return () => {
      chartRef.current?.destroy();
      chartRef.current = undefined;
    };
  }, [ config, dimensions ]);

  useEffect(() => {
    let redraw = true;

    try {
      chartRef.current?.setDimensions(dimensions, false);
    } catch (e) {
      redraw = false;
    }

    try {
      if (config) chartRef.current?.setConfig(config, false);
    } catch (e) {
      redraw = false;
    }

    try {
      chartRef.current?.setData(data, false);
    } catch (e) {
      redraw = false;
    }

    if (redraw) chartRef.current?.redraw();
  }, [ config, data, dimensions ]);

  return (
    <div className={css.base}>
      <div className={css.note}>
        Click and drag along the axes to create filters.
        Click on existing filters to remove them.
        Double click to reset.
      </div>
      <div ref={containerRef} style={{ height: `${height}px` }} />
    </div>
  );
};

export default ParallelCoordinates;
