import Hermes from 'hermes-parallel-coordinates';
import React, { useEffect, useRef } from 'react';

import useTheme from 'hooks/useTheme';
import { getCssVar } from 'themes';

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
  const { mode } = useTheme();

  useEffect(() => {
    if (!containerRef.current) return;

    chartRef.current = new Hermes(containerRef.current);

    return () => {
      chartRef.current?.destroy();
      chartRef.current = undefined;
    };
  }, [ dimensions ]);

  useEffect(() => {
    let redraw = true;

    try {
      chartRef.current?.setDimensions(dimensions, false);
    } catch (e) {
      redraw = false;
    }

    try {
      if (config) {
        const newConfig = Hermes.deepMerge({
          style: {
            axes: {
              label: {
                fillStyle: getCssVar('--theme-surface-on'),
                strokeStyle: getCssVar('--theme-surface-weak'),
              },
              labelActive: {
                fillStyle: getCssVar('--theme-surface-on-strong'),
                strokeStyle: getCssVar('--theme-surface-weak'),
              },
              labelHover: {
                fillStyle: getCssVar('--theme-surface-on-strong'),
                strokeStyle: getCssVar('--theme-surface-weak'),
              },
            },
            dimension: {
              label: {
                fillStyle: getCssVar('--theme-surface-on'),
                strokeStyle: getCssVar('--theme-surface-weak'),
              },
              labelActive: {
                fillStyle: getCssVar('--theme-status-active'),
                strokeStyle: getCssVar('--theme-surface-weak'),
              },
              labelHover: {
                fillStyle: getCssVar('--theme-status-active'),
                strokeStyle: getCssVar('--theme-surface-weak'),
              },
            },
          },
        }, config);
        chartRef.current?.setConfig(newConfig, false);
      }
    } catch (e) {
      redraw = false;
    }

    try {
      chartRef.current?.setData(data, false);
    } catch (e) {
      redraw = false;
    }

    if (redraw) chartRef.current?.redraw();
  }, [ config, data, dimensions, mode ]);

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
