import Hermes from 'hermes-parallel-coordinates';
import React, { useEffect, useRef } from 'react';

import css from 'components/ParallelCoordinates.module.scss';
import useUI from 'stores/contexts/UI';

interface Props {
  config?: Hermes.RecursivePartial<Hermes.Config>;
  data: Hermes.Data;
  dimensions: Hermes.Dimension[];
  disableInteraction?: boolean;
  height?: number;
}

const ParallelCoordinates: React.FC<Props> = ({
  config,
  data,
  dimensions,
  disableInteraction = false,
  height = 450,
}: Props) => {
  const chartRef = useRef<Hermes>();
  const containerRef = useRef<HTMLDivElement>(null);
  const { ui } = useUI();

  useEffect(() => {
    if (!containerRef.current) return;
    chartRef.current = new Hermes(containerRef.current);

    return () => {
      chartRef.current?.destroy();
      chartRef.current = undefined;
    };
  }, []);

  useEffect(() => {
    if (disableInteraction) chartRef.current?.disable();
  }, [disableInteraction]);

  useEffect(() => {
    let redraw = true;

    try {
      chartRef.current?.setDimensions(dimensions, false);
    } catch (e) {
      redraw = false;
    }

    try {
      if (config) {
        const newConfig = Hermes.deepMerge(config, {
          style: {
            axes: {
              label: {
                fillStyle: ui.theme.surfaceOn,
                strokeStyle: ui.theme.surfaceWeak,
              },
              labelActive: {
                fillStyle: ui.theme.surfaceOnStrong,
                strokeStyle: ui.theme.surfaceWeak,
              },
              labelHover: {
                fillStyle: ui.theme.surfaceOnStrong,
                strokeStyle: ui.theme.surfaceWeak,
              },
            },
            dimension: {
              label: {
                fillStyle: ui.theme.surfaceOn,
                strokeStyle: ui.theme.surfaceWeak,
              },
              labelActive: {
                fillStyle: ui.theme.statusActive,
                strokeStyle: ui.theme.surfaceWeak,
              },
              labelHover: {
                fillStyle: ui.theme.statusActive,
                strokeStyle: ui.theme.surfaceWeak,
              },
            },
          },
        });
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
  }, [config, data, dimensions, ui.theme]);

  return (
    <div className={css.base}>
      {!disableInteraction && (
        <div className={css.note}>
          Click and drag along the axes to create filters. Click on existing filters to remove them.
          Double click to reset.
        </div>
      )}
      <div ref={containerRef} style={{ height: `${height}px` }} />
    </div>
  );
};

export default ParallelCoordinates;
