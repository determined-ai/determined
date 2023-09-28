import { DataEditorProps, GridMouseEventArgs } from '@hpe.com/glide-data-grid';
import { Tooltip } from 'antd';
import React, { ReactNode, useCallback, useEffect, useMemo } from 'react';

import { Loadable } from 'components/kit/utils/loadable';

import { ColumnDefs } from './columns';
import { GlideTableProps } from './GlideTable';

export interface TooltipProps {
  x: number;
  y: number;
  open: boolean;
  text: string;
}

interface UseTooltipParams {
  data: GlideTableProps['data'];
  columnIds: string[];
  columnDefs: ColumnDefs;
}
interface UseTooltipReturn {
  closeTooltip: () => void;
  onItemHovered: DataEditorProps['onItemHovered'];
  tooltip: ReactNode;
}

export const useTableTooltip = ({
  data,
  columnIds,
  columnDefs,
}: UseTooltipParams): UseTooltipReturn => {
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
          const columnId = columnIds[columnIdx];
          const text = columnDefs?.[columnId]?.tooltip(record.data);
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
    [data, columnIds, columnDefs, closeTooltip],
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
          <Tooltip open={tooltipProps.open} placement="top" title={tooltipProps.text} />
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
