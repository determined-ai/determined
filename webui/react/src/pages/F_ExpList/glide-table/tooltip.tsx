import { DataEditorProps, GridMouseEventArgs } from '@glideapps/glide-data-grid';
import Tooltip from 'hew/Tooltip';
import { Loadable } from 'hew/utils/loadable';
import React, { ReactNode, useCallback, useEffect, useMemo } from 'react';

import { ColumnDef } from './columns';

export interface TooltipProps {
  x: number;
  y: number;
  open: boolean;
  text: string;
}

interface UseTooltipParams<T> {
  data: Loadable<T>[];
  columns: ColumnDef<T>[];
}
interface UseTooltipReturn {
  closeTooltip: () => void;
  onItemHovered: DataEditorProps['onItemHovered'];
  tooltip: ReactNode;
}

export function useTableTooltip<T>({
  columns,
  data,
}: UseTooltipParams<T>): UseTooltipReturn {
  const [tooltipProps, setTooltipProps] = React.useState<TooltipProps | undefined>(undefined);

  const timeoutRef = React.useRef(0);
  const closeTooltip = useCallback(() => {
    window.clearTimeout(timeoutRef.current);
    timeoutRef.current = 0;
    setTooltipProps(undefined);
  }, []);

  const onItemHovered: DataEditorProps['onItemHovered'] = React.useCallback(
    (args: GridMouseEventArgs) => {
      if (args.kind === 'cell') {
        window.clearTimeout(timeoutRef.current);
        setTooltipProps(undefined);
        const [columnIdx, rowIdx] = args.location;
        const record = data[rowIdx];
        if (record && Loadable.isLoaded(record)) {
          const text = columns[columnIdx]?.tooltip(record.data);
          if (text) {
            timeoutRef.current = window.setTimeout(() => {
              setTooltipProps({
                open: true,
                text: text,
                x: args.bounds.x + 20,
                y: args.bounds.y + 20,
              });
            }, 500);
          }
        }
      } else {
        closeTooltip();
      }
    },
    [data, columns, closeTooltip],
  );

  useEffect(() => () => window.clearTimeout(timeoutRef.current), []);

  const tooltip = useMemo(
    () =>
      tooltipProps ? (
        <div
          style={{
            left: tooltipProps.x,
            position: 'fixed',
            top: tooltipProps.y,
            zIndex: 10,
          }}>
          <Tooltip content={tooltipProps.text} open={tooltipProps.open} placement="top" />
        </div>
      ) : null,
    [tooltipProps],
  );

  return {
    closeTooltip,
    onItemHovered,
    tooltip,
  };
};
