import { Table } from 'antd';
import { SpinProps } from 'antd/es/spin';
import { TableProps } from 'antd/es/table';
import { ColumnsType, ColumnType, SorterResult } from 'antd/es/table/interface';
import React, {
  createContext,
  CSSProperties,
  MutableRefObject,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useDrag, useDragLayer, useDrop } from 'react-dnd';
import {
  DraggableCore,
  DraggableData,
  DraggableEvent,
  DraggableEventHandler,
} from 'react-draggable';

import SkeletonTable from 'components/Table/SkeletonTable';
import useResize from 'hooks/useResize';
import { UpdateSettings } from 'hooks/useSettings';
import Spinner from 'shared/components/Spinner/Spinner';
import { Primitive, UnknownRecord } from 'shared/types';
import { isEqual } from 'shared/utils/data';

import css from './InteractiveTable.module.scss';

/*
 * This indicates that the cell contents are rightClickable
 * and we should disable custom context menu on cell context hover
 */
export const onRightClickableCell = (): React.HTMLAttributes<HTMLElement> =>
  ({ isCellRightClickable: true } as React.HTMLAttributes<HTMLElement>);

export interface InteractiveTableSettings {
  /**
   * ColumnWidths: Array of column widths, corresponding to columns array below
   */
  columnWidths: number[];
  /**
   * Columns: Array of column names
   */
  columns: string[];
  /**
   * Row: Array of selected row IDs
   */
  row?: number[] | string[];
  sortDesc: boolean;
  sortKey?: Primitive;
  tableLimit: number;
  tableOffset: number;
}

export const WIDGET_COLUMN_WIDTH = 46;
const DEFAULT_RESIZE_THROTTLE_TIME = 30;
const SOURCE_TYPE = 'DraggableColumn';

type DndItem = {
  index?: number;
};
export interface ContextMenuProps<RecordType> {
  children: React.ReactNode;
  onVisibleChange: (visible: boolean) => void;
  record: RecordType;
}

export interface ColumnDef<RecordType> extends ColumnType<RecordType> {
  dataIndex: string;
  defaultWidth: number;
  isFiltered?: (s: unknown) => boolean;
}
export type ColumnDefs<ColumnName extends string, RecordType> = Record<
  ColumnName,
  ColumnDef<RecordType>
>;

interface InteractiveTableProps<RecordType> extends TableProps<RecordType> {
  ContextMenu?: React.FC<ContextMenuProps<RecordType>>;
  areRowsSelected?: boolean;
  columns: ColumnDef<RecordType>[];
  containerRef: MutableRefObject<HTMLElement | null>;
  interactiveColumns?: boolean;
  numOfPinned?: number;
  settings: InteractiveTableSettings;
  updateSettings: UpdateSettings<InteractiveTableSettings>;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
type InteractiveTable = <T extends object>(props: InteractiveTableProps<T>) => JSX.Element;
type Row = <T extends object>(props: RowProps<T>) => JSX.Element;

type DragState = 'draggingRight' | 'draggingLeft' | 'notDragging';
interface RowProps<RecordType> {
  ContextMenu?: React.FC<ContextMenuProps<RecordType>>;
  areRowsSelected?: boolean;
  children?: React.ReactNode;
  className?: string;
  index: number;
  numOfPinned?: number;
  record: RecordType;
}

interface HeaderCellProps {
  className: string;
  columnName: string;
  dragState: DragState;
  dropLeftStyle: CSSProperties;
  dropRightStyle: CSSProperties;
  filterActive: boolean;
  index: number;
  interactiveColumns: boolean;
  isResizing: boolean;
  minWidth: number;
  moveColumn: (source: number, destination: number) => void;
  onResize: (e: DraggableEvent, data: DraggableData) => number;
  onResizeStart: DraggableEventHandler;
  onResizeStop: DraggableEventHandler;
  title: unknown;
  width: number;
}

interface CellProps {
  children?: React.ReactNode;
  className: string;
  isCellRightClickable?: boolean;
}

interface RightClickableCellProps {
  onContextMenu: (e: React.MouseEvent) => void;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}

const RightClickableRowContext = createContext({});

const getAdjustedColumnWidthSum = (columnsWidths: number[]) => {
  return columnsWidths.reduce((a, b) => a + b, 0) + 2 * WIDGET_COLUMN_WIDTH + 2 * 24;
};

const Cell = ({ children, className, isCellRightClickable, ...props }: CellProps) => {
  const rightClickableCellProps = useContext(RightClickableRowContext);
  const classes = [className, css.cell];
  if (!isCellRightClickable)
    return (
      <td className={classes.join(' ')} {...props}>
        {children}
      </td>
    );
  return (
    <td className={classes.join(' ')} {...props}>
      <div className={css.rightClickableCellWrapper} {...rightClickableCellProps}>
        {children}
      </div>
    </td>
  );
};

export const Row: Row = ({
  className,
  children,
  record,
  ContextMenu,
  areRowsSelected,
  index,
  numOfPinned,
  ...props
}) => {
  const classes = [className, css.row];

  const [rowHovered, setRowHovered] = useState(false);
  const [rightClickableCellHovered, setRightClickableCellHovered] = useState(false);
  const [contextMenuOpened, setContextMenuOpened] = useState(false);
  const isPinned = Array.from(Array(numOfPinned).keys()).includes(index);

  const rightClickableCellProps: RightClickableCellProps = {
    onContextMenu: (e: React.MouseEvent) => e.stopPropagation(),
    onMouseEnter: () => {
      if (!rightClickableCellHovered) setRightClickableCellHovered(true);
    },
    onMouseLeave: () => {
      if (rightClickableCellHovered) setRightClickableCellHovered(false);
    },
  };

  if (areRowsSelected) {
    return (
      <tr className={classes.join(' ')} {...props}>
        {children}
      </tr>
    );
  }

  const rowContextMenuTriggerableOrOpen =
    (rowHovered && !rightClickableCellHovered) || contextMenuOpened;

  if (rowContextMenuTriggerableOrOpen) {
    classes.push('ant-table-row-selected');
  }

  if (isPinned && numOfPinned === index + 1) {
    classes.push(css.lastPinnedRow);
  }

  return record && ContextMenu ? (
    <RightClickableRowContext.Provider value={{ ...rightClickableCellProps }}>
      <ContextMenu record={record} onVisibleChange={setContextMenuOpened}>
        <tr
          className={classes.join(' ')}
          onMouseEnter={() => {
            if (!rowHovered) setRowHovered(true);
          }}
          onMouseLeave={() => {
            if (rowHovered) setRowHovered(false);
          }}
          {...props}
          style={isPinned ? { position: 'sticky', top: 60 * index, zIndex: 10 } : undefined}>
          {children}
        </tr>
      </ContextMenu>
    </RightClickableRowContext.Provider>
  ) : (
    <tr className={classes.join(' ')} {...props}>
      {children}
    </tr>
  );
};

const ResizeShadow: React.FC<{ display: 'none' | 'block'; x: number }> = React.memo(
  ({ x, display }) => {
    return <span className={css.resizeShadow} style={{ display, left: x, position: 'absolute' }} />;
  },
);

const HeaderCell = ({
  onResize,
  onResizeStart,
  onResizeStop,
  width,
  className,
  columnName,
  filterActive,
  minWidth,
  moveColumn,
  index,
  title: unusedTitleFromAntd,
  isResizing,
  interactiveColumns,
  dropRightStyle,
  dropLeftStyle,
  dragState,
  ...props
}: HeaderCellProps) => {
  const resizingRef = useRef<HTMLDivElement>(null);
  const [xValue, setXValue] = useState(0);
  const [shadowVisibility, setShadowVisibility] = useState<'none' | 'block'>('none');

  const headerCellClasses = [css.headerCell];
  const dropTargetClasses = [css.dropTarget];

  const [, drag] = useDrag({
    canDrag: () => !isResizing,
    item: { index },
    type: SOURCE_TYPE,
  });

  const [{ isOver }, drop] = useDrop({
    accept: SOURCE_TYPE,
    canDrop: (_, monitor) => {
      const dragItem = monitor.getItem() || {};
      const dragIndex = dragItem?.index;
      const deltaX = monitor.getDifferenceFromInitialOffset()?.x;
      const internalDragState = (() => {
        if (!deltaX) return 'notDragging';

        if (deltaX > 0) return 'draggingRight';

        return 'draggingLeft';
      })();
      if (
        dragIndex == null ||
        dragIndex === index ||
        (internalDragState === 'draggingRight' && dragIndex > index) ||
        (internalDragState === 'draggingLeft' && dragIndex < index)
      ) {
        return false;
      }
      return true;
    },
    collect: (monitor) => {
      return { isOver: monitor.canDrop() && monitor.isOver() };
    },
    drop: (item: DndItem) => {
      if (item.index != null) {
        moveColumn(item.index, index);
      }
    },
  });

  if (isOver) dropTargetClasses.push(css.dropTargetActive);
  if (filterActive) headerCellClasses.push(css.headerFilterOn);

  if (!columnName) return <th className={className} {...props} />;

  if (!interactiveColumns) return <th className={headerCellClasses.join(' ')} {...props} />;

  const tableCell = (
    <th className={headerCellClasses.join(' ')}>
      <div
        className={`${className} ${css.columnDraggingDiv}`}
        ref={drag}
        title={columnName}
        onClick={(e) => e.stopPropagation()}
        {...props}
      />
      <DraggableCore
        nodeRef={resizingRef}
        onDrag={(e, data) => {
          onResize(e, data);
          const newWidth = data.x < minWidth ? minWidth : data.x;

          if (newWidth !== xValue) setXValue(newWidth);
        }}
        onStart={(e, data) => {
          setShadowVisibility('block');
          const newWidth = data.x < minWidth ? minWidth : data.x;
          if (newWidth !== xValue) setXValue(newWidth);
          onResizeStart(e, data);
          setShadowVisibility('block');
        }}
        onStop={(e, data) => {
          onResizeStop(e, data);
          setShadowVisibility('none');
        }}>
        <span
          className={css.columnResizeHandle}
          ref={resizingRef}
          onClick={(e) => e.stopPropagation()}
        />
      </DraggableCore>
      <ResizeShadow display={shadowVisibility} x={xValue} />
      <span
        className={dropTargetClasses.join(' ')}
        ref={drop}
        style={
          dragState === 'draggingRight'
            ? dropRightStyle
            : dragState === 'draggingLeft'
            ? dropLeftStyle
            : {}
        }
      />
      <span
        className={isOver ? css.dropTargetIndicator : ''}
        style={
          dragState === 'draggingRight'
            ? { right: '-3px' }
            : dragState === 'draggingLeft'
            ? { left: '-3px' }
            : {}
        }
      />
    </th>
  );
  return tableCell;
};

const InteractiveTable: InteractiveTable = ({
  loading,
  scroll,
  dataSource,
  columns,
  containerRef,
  interactiveColumns = true,
  numOfPinned,
  settings,
  updateSettings,
  ContextMenu,
  areRowsSelected,
  ...props
}) => {
  const columnDefs = useMemo(
    () =>
      columns
        ?.map((col) => ({ [col.dataIndex as string]: col }))
        .reduce((a, b) => ({ ...a, ...b })),
    [columns],
  ) as ColumnDefs<string, UnknownRecord>;
  const { width: pageWidth } = useResize(containerRef);
  const tableRef = useRef<HTMLDivElement>(null);
  const timeout = useRef<NodeJS.Timeout>();

  const getUpscaledWidths = useCallback(
    (widths: number[]): number[] => {
      let newWidths = widths;
      if (pageWidth) {
        const sumOfWidths = getAdjustedColumnWidthSum(newWidths);
        const shortage = pageWidth - sumOfWidths;
        if (shortage > 0) {
          const compensatingPortion = shortage / newWidths.length;
          newWidths = newWidths.map((w) => w + compensatingPortion);
        }
      }
      return newWidths.map(Math.round);
    },
    [pageWidth],
  );

  const [widthData, setWidthData] = useState(() => {
    const widths = settings?.columnWidths || [];
    return {
      dropLeftStyles:
        widths.map((width, idx) => ({
          left: `${-((widths[idx - 1] ?? WIDGET_COLUMN_WIDTH) / 2)}px`,
          width: `${(width + (widths[idx - 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
        })) ?? [],
      dropRightStyles:
        widths.map((width, idx) => ({
          left: `${width / 2}px`,
          width: `${(width + (widths[idx + 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
        })) ?? [],
      widths: widths ?? [],
    };
  });

  const [isResizing, setIsResizing] = useState(false);

  const { dragState } = useDragLayer((monitor) => {
    const deltaX = monitor.getDifferenceFromInitialOffset()?.x;
    const dragState = (() => {
      if (!deltaX) return 'notDragging';

      if (deltaX > 0) return 'draggingRight';

      return 'draggingLeft';
    })();
    return { dragState };
  });

  const spinning = !!(loading as SpinProps)?.spinning || loading === true;

  const handleChange = useCallback(
    (tablePagination, tableFilters, tableSorter): void => {
      if (Array.isArray(tableSorter)) return;

      const newSettings: Partial<InteractiveTableSettings> = {
        tableLimit: tablePagination.pageSize,
        tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
      };

      const { columnKey, order } = tableSorter as SorterResult<unknown>;
      if (columnKey && settings.columns.find((col) => columnDefs[col]?.key === columnKey)) {
        newSettings.sortDesc = order === 'descend';
        newSettings.sortKey = columnKey;
      }

      const shouldPush = settings.tableOffset !== newSettings.tableOffset;

      if (isEqual(newSettings, settings)) return;

      updateSettings(newSettings, shouldPush);
    },
    [settings, updateSettings, columnDefs],
  );

  const moveColumn = useCallback(
    (fromIndex, toIndex) => {
      const reorderedColumns = [...settings.columns];
      const reorderedWidths = [...settings.columnWidths];
      const col = reorderedColumns.splice(fromIndex, 1)[0];
      const width = reorderedWidths.splice(fromIndex, 1)[0];
      reorderedColumns.splice(toIndex, 0, col);
      reorderedWidths.splice(toIndex, 0, width);
      updateSettings({ columns: reorderedColumns, columnWidths: reorderedWidths });
      setWidthData({ ...widthData, widths: reorderedWidths });
    },
    [settings.columns, settings.columnWidths, widthData, updateSettings],
  );

  const handleResize = useCallback(
    (resizeIndex) => {
      return (e: Event, { x }: DraggableData) => {
        if (timeout.current) clearTimeout(timeout.current);
        const column = settings.columns[resizeIndex];
        const minWidth = columnDefs[column]?.defaultWidth ?? 40;
        const currentWidths = widthData.widths;

        if (x === currentWidths[resizeIndex]) return;

        let targetWidths = currentWidths;

        targetWidths[resizeIndex] = x < minWidth ? minWidth : x;

        const targetWidthSum = getAdjustedColumnWidthSum(targetWidths);
        /**
         * If the table width is less than the page width, the browser upscales,
         * and then the resize no longer tracks with the cursor.
         * we manually do the scaling here to keep the tableWidth >= pageWidth
         * in particular, we distribute the deficit among the other columns
         */
        const shortage = pageWidth - targetWidthSum;
        if (shortage > 0) {
          const compensatingPortion = shortage / (currentWidths.length - 1);
          targetWidths = targetWidths.map((targetWidth, targetWidthIndex) =>
            targetWidthIndex === resizeIndex ? targetWidth : targetWidth + compensatingPortion,
          );
        }

        const dropRightStyles = targetWidths.map((width, idx) => ({
          left: `${width / 2}px`,
          width: `${(width + (targetWidths[idx + 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
        }));
        const dropLeftStyles = targetWidths.map((width, idx) => ({
          left: `${-((targetWidths[idx - 1] ?? WIDGET_COLUMN_WIDTH) / 2)}px`,
          width: `${(width + (targetWidths[idx - 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
        }));

        timeout.current = setTimeout(() => {
          setWidthData({ dropLeftStyles, dropRightStyles, widths: targetWidths });
        }, DEFAULT_RESIZE_THROTTLE_TIME);
      };
    },
    [settings.columns, widthData.widths, pageWidth, columnDefs],
  );

  const handleResizeStart = useCallback(
    (index) =>
      (e: Event, { x }: DraggableData) => {
        setIsResizing(true);

        setWidthData(({ widths, ...rest }) => {
          const column = settings.columns[index];
          const startWidth = widths[index];
          const minWidth = columnDefs[column]?.defaultWidth ?? 40;
          const deltaX = startWidth - minWidth;
          const minX = x - deltaX;
          return { minX, widths, ...rest };
        });
      },
    [settings.columns, columnDefs],
  );

  const handleResizeStop = useCallback(() => {
    setIsResizing(false);

    updateSettings({ columnWidths: widthData.widths.map(Math.round) });
  }, [updateSettings, widthData]);

  const onHeaderCell = useCallback(
    (index, columnDef) => {
      return () => {
        const filterActive = !!columnDef?.isFiltered?.(settings);
        return {
          columnName: columnDef.title,
          dragState,
          dropLeftStyle: { ...widthData?.dropLeftStyles?.[index] },
          dropRightStyle: { ...widthData?.dropRightStyles?.[index] },
          filterActive,
          index,
          interactiveColumns,
          isResizing,
          minWidth: columnDef.defaultWidth,
          moveColumn,
          onResize: handleResize(index),
          onResizeStart: handleResizeStart(index),
          onResizeStop: handleResizeStop,
          width: widthData?.widths[index] ?? columnDef.defaultWidth,
        };
      };
    },
    [
      handleResize,
      handleResizeStop,
      widthData,
      moveColumn,
      settings,
      handleResizeStart,
      interactiveColumns,
      dragState,
      isResizing,
    ],
  );

  const renderColumns = useMemo(() => {
    const columns = settings.columns.reduce<ColumnsType<UnknownRecord>>(
      (acc, columnName, index) => {
        if (!columnDefs[columnName]) return acc;

        const column = columnDefs[columnName];
        const currentWidth = widthData.widths[index];
        const columnWidth = currentWidth < column.defaultWidth ? column.defaultWidth : currentWidth; // avoid rendering a column with less width than the default
        const sortOrder =
          column.key === settings.sortKey ? (settings.sortDesc ? 'descend' : 'ascend') : null;

        acc.push({
          onHeaderCell: onHeaderCell(index, column),
          sortOrder,
          width: columnWidth,
          ...column,
        });

        return acc;
      },
      [],
    );

    if (columnDefs.action) {
      columns.push({ ...columnDefs.action, width: WIDGET_COLUMN_WIDTH });
    }

    return columns;
  }, [settings.columns, widthData, settings.sortKey, settings.sortDesc, columnDefs, onHeaderCell]);

  const components = {
    body: {
      cell: Cell,
      row: Row,
    },
    header: { cell: HeaderCell },
  };

  useEffect(() => {
    return () => {
      if (timeout.current) clearTimeout(timeout.current);
    };
  }, []);

  useEffect(() => {
    // this should run only when getting new number of cols
    if (settings.columnWidths.length === widthData.widths.length) return;

    const widths = getUpscaledWidths(settings.columnWidths);
    const dropRightStyles = widths.map((width, idx) => ({
      left: `${width / 2}px`,
      width: `${(width + (widths[idx + 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
    }));
    const dropLeftStyles = widths.map((width, idx) => ({
      left: `${-((widths[idx - 1] ?? WIDGET_COLUMN_WIDTH) / 2)}px`,
      width: `${(width + (widths[idx - 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
    }));

    setWidthData({ dropLeftStyles, dropRightStyles, widths });
  }, [settings.columnWidths, widthData, getUpscaledWidths]);

  return (
    <div className={css.tableContainer} ref={tableRef}>
      <Spinner spinning={spinning}>
        {spinning ? (
          <SkeletonTable columns={renderColumns.length} />
        ) : (
          <Table
            bordered
            /* next one is just so ant doesnt complain */
            /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
            columns={renderColumns as ColumnsType<any>}
            components={components}
            dataSource={dataSource}
            scroll={scroll}
            tableLayout="fixed"
            onChange={handleChange}
            onRow={(record, index) =>
              ({
                areRowsSelected,
                ContextMenu,
                index,
                numOfPinned,
                record,
              } as React.HTMLAttributes<HTMLElement>)
            }
            {...props}
          />
        )}
      </Spinner>
    </div>
  );
};

export default InteractiveTable;
