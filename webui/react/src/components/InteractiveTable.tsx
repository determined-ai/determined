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
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import 'antd/dist/antd.min.css';
import { useDrag, useDragLayer, useDrop } from 'react-dnd';
import { DraggableCore, DraggableData, DraggableEventHandler } from 'react-draggable';
import { throttle } from 'throttle-debounce';

import useResize from 'hooks/useResize';
import { DEFAULT_COLUMN_WIDTHS, Settings } from 'pages/ExperimentList.settings';
import { ExperimentItem } from 'types';

import css from './InteractiveTable.module.scss';
import Spinner from './Spinner';

const WIDGET_COLUMN_WIDTH = 46;
const DEFAULT_RESIZE_THROTTLE_TIME = 10;

const type = 'DraggableColumn';

type DndItem = {
  index?: number;
}
interface ContextMenuProps {
  onVisibleChange: (visible: boolean) => void;
  record: Record<string, unknown>;
}

interface ColumnDef<RecordType> extends ColumnType<RecordType> {
  isFiltered?: (s: Settings) => boolean;
}
export type ColumnDefs<ColumnName extends string, RecordType> = Record<
  ColumnName,
  ColumnDef<RecordType>
>;

interface InteractiveTableProps<RecordType> extends TableProps<RecordType> {
  ContextMenu?: React.FC<ContextMenuProps>;
  areRowsSelected?: boolean;
  columnDefs: ColumnDefs<string, RecordType>;
  containerRef: MutableRefObject<HTMLElement | null>,
  settings: Settings;
  updateSettings: (settings: Partial<Settings>, shouldPush?: boolean) => void;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
type InteractiveTable = <T extends object>(props: InteractiveTableProps<T>) => JSX.Element;

type DragState = 'draggingRight' | 'draggingLeft' | 'notDragging';
interface RowProps {
  ContextMenu: React.FC<ContextMenuProps>;
  areRowsSelected?: boolean;
  children?: React.ReactNode;
  className?: string;
  record: Record<string, unknown>;
}

interface HeaderCellProps {
  className: string;
  columnName: string;
  dragState : DragState;
  dropLeftStyle: CSSProperties;
  dropRightStyle: CSSProperties;
  filterActive: boolean;
  index: number;
  isResizing: boolean;
  moveColumn: (source: number, destination: number) => void;
  onResize: DraggableEventHandler;
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

const RightClickableRowContext = createContext({});

const Row = ({
  className,
  children,
  record,
  ContextMenu,
  areRowsSelected,
  ...props
}: RowProps) => {
  const classes = [ className, css.row ];

  const [ rowHovered, setRowHovered ] = useState(false);
  const [ rightClickableCellHovered, setRightClickableCellHovered ] = useState(false);
  const [ contextMenuOpened, setContextMenuOpened ] = useState(false);

  if (areRowsSelected) {
    return <tr className={classes.join(' ')} {...props}>{children}</tr>;
  }

  const rightClickableCellProps = {
    onContextMenu: (e : React.MouseEvent) => e.stopPropagation(),
    onMouseEnter: () => setRightClickableCellHovered(true),
    onMouseLeave: () => setRightClickableCellHovered(false),
  };

  const rowContextMenuTriggerableOrOpen =
    (rowHovered && !rightClickableCellHovered) || contextMenuOpened;

  if (rowContextMenuTriggerableOrOpen) {
    classes.push('ant-table-row-selected');
  }
  return record ? (
    <RightClickableRowContext.Provider value={{ ...rightClickableCellProps }}>
      <ContextMenu record={record} onVisibleChange={setContextMenuOpened}>
        <tr
          className={
            classes.join(' ')
          }
          onMouseEnter={() => setRowHovered(true)}
          onMouseLeave={() => setRowHovered(false)}
          {...props}>
          {children}
        </tr>
      </ContextMenu>
    </RightClickableRowContext.Provider>
  ) : (
    <tr className={classes.join(' ')} {...props}>{children}</tr>
  );
};

const Cell = ({ children, className, isCellRightClickable, ...props }: CellProps) => {
  const rightClickableCellProps = useContext(RightClickableRowContext);
  const classes = [ className, css.cell ];
  if (!isCellRightClickable) return <td className={classes.join(' ')} {...props}>{children}</td>;
  return (
    <td className={classes.join(' ')} {...props}>
      <div className={css.rightClickableCellWrapper} {...rightClickableCellProps}>
        {children}
      </div>
    </td>
  );
};

const HeaderCell = ({
  onResize,
  onResizeStart,
  onResizeStop,
  width,
  className,
  columnName,
  filterActive,
  moveColumn,
  index,
  title: unusedTitleFromAntd,
  isResizing,
  dropRightStyle,
  dropLeftStyle,
  dragState,
  ...props
}: HeaderCellProps) => {
  const resizingRef = useRef<HTMLDivElement>(null);

  const headerCellClasses = [ css.headerCell ];
  const dropTargetClasses = [ css.dropTarget ];
  const [ , drag ] = useDrag({
    canDrag: () => !isResizing,
    item: { index },
    type,
  });

  const [ { isOver }, drop ] = useDrop({
    accept: type,
    canDrop: (_, monitor) => {
      const dragItem = (monitor.getItem() || {}); // as DndItem;
      const dragIndex = dragItem?.index;
      const deltaX = monitor.getDifferenceFromInitialOffset()?.x;
      const dragState = deltaX ? (deltaX > 0 ? 'draggingRight' : 'draggingLeft') : 'notDragging';
      if (
        dragIndex == null ||
        dragIndex === index ||
        (dragState === 'draggingRight' && dragIndex > index) ||
        (dragState === 'draggingLeft' && dragIndex < index)
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

  if (isOver) {
    dropTargetClasses.push(css.dropTargetActive);
  }

  if (filterActive) headerCellClasses.push(css.headerFilterOn);

  if (!columnName) {
    return <th className={className} {...props} />;
  }

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
        onDrag={onResize}
        onStart={onResizeStart}
        onStop={onResizeStop}>
        <span
          className={css.columnResizeHandle}
          ref={resizingRef}
          onClick={(e) => {
            e.stopPropagation();
          }}
        />
      </DraggableCore>
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
  columnDefs,
  containerRef,
  settings,
  updateSettings,
  ContextMenu,
  areRowsSelected,
  ...props
}) => {
  const { width: pageWidth } = useResize(containerRef);
  const tableRef = useRef<HTMLDivElement>(null);
  const [ widthData, setWidthData ] = useState({
    dropLeftStyles: settings?.columnWidths?.map(() => ({}) ?? []),
    dropRightStyles: settings?.columnWidths?.map(() => ({}) ?? []),
    widths: settings?.columnWidths,
  });
  const [ isResizing, setIsResizing ] = useState(false);

  const { dragState } = useDragLayer((monitor) => {
    const deltaX = monitor.getDifferenceFromInitialOffset()?.x;
    const dragState = deltaX ? (deltaX > 0 ? 'draggingRight' : 'draggingLeft') : 'notDragging';
    return { dragState };
  });

  const spinning = !!(loading as SpinProps)?.spinning || loading === true;

  const adjustedColumnWidthSum = useCallback((columnsWidths: number[]) => {
    const pagePadding = parseInt(
      getComputedStyle(document.body)
        ?.getPropertyValue('--theme-sizes-layout-big').slice(0, -2),
    ) ?? 16;
    return columnsWidths.reduce((a, b) => a + b) + 2 * WIDGET_COLUMN_WIDTH + 2 * pagePadding;
  }, []);

  const getUpscaledWidths = useCallback(
    (widths: number[]): number[] => {
      let newWidths = widths;
      if (pageWidth) {
        const sumOfWidths = adjustedColumnWidthSum(newWidths);
        if (sumOfWidths < pageWidth) {
          const scaleUp = pageWidth / sumOfWidths;
          newWidths = widths.map((w: number) => w * scaleUp);
        }
      }
      return newWidths.map(Math.floor);
    },
    [ pageWidth, adjustedColumnWidthSum ],
  );

  useLayoutEffect(() => {
    const prevWidths = settings.columnWidths;
    const prevSumOfWidths = adjustedColumnWidthSum(prevWidths);
    const widths = getUpscaledWidths(prevWidths);
    const sumOfWidths = adjustedColumnWidthSum(widths);

    const dropRightStyles = widths.map((w, i) => ({
      left: `${(w / 2) }px`,
      width: `${(w + (widths[i + 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
    }));
    const dropLeftStyles = widths.map((w, i) => ({
      left: `${-((widths[i - 1] ?? WIDGET_COLUMN_WIDTH) / 2)}px`,
      width: `${(w + (widths[i - 1] ?? WIDGET_COLUMN_WIDTH)) / 2}px`,
    }));
    setWidthData({ dropLeftStyles, dropRightStyles, widths });
    if (sumOfWidths !== prevSumOfWidths) {
      updateSettings({ columnWidths: widths });
    }
  }, [ settings.columnWidths,
    getUpscaledWidths,
    updateSettings,
    pageWidth,
    adjustedColumnWidthSum ]);

  const handleChange = useCallback(
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    (tablePagination: any, tableFilters: any, tableSorter: any): void => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<unknown>;
      if (!columnKey || !settings.columns.find((col) => columnDefs[col]?.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
        sortKey: columnKey as any,
        tableLimit: tablePagination.pageSize,
        tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
      };
      const shouldPush = settings.tableOffset !== newSettings.tableOffset;
      updateSettings(newSettings, shouldPush);
    },
    [ settings, updateSettings, columnDefs ],
  );

  const moveColumn = useCallback(
    (fromIndex, toIndex) => {
      const reorderedColumns = [ ...settings.columns ];
      const reorderedWidths = [ ...settings.columnWidths ];
      const col = reorderedColumns.splice(fromIndex, 1)[0];
      const width = reorderedWidths.splice(fromIndex, 1)[0];
      reorderedColumns.splice(toIndex, 0, col);
      reorderedWidths.splice(toIndex, 0, width);
      updateSettings({ columns: reorderedColumns, columnWidths: reorderedWidths });
    },
    [ settings.columns, settings.columnWidths, updateSettings ],
  );

  const handleResize = useCallback(
    (index) => {
      return throttle(DEFAULT_RESIZE_THROTTLE_TIME, (e: Event, { x }: DraggableData) => {
        setWidthData(({ widths: prevWidths, ...rest }) => {
          const column = settings.columns[index];
          const minWidth = DEFAULT_COLUMN_WIDTHS[column];
          let targetWidths;
          if (x < minWidth) {
            targetWidths = settings.columnWidths.map((w: number, i: number) =>
              index === i ? minWidth : w);

          } else {
            const newWidth = x;
            targetWidths = settings.columnWidths.map((w: number, i: number) =>
              index === i ? newWidth : w);
          }

          const targetWidthSum = adjustedColumnWidthSum(targetWidths);
          const shortage = pageWidth - targetWidthSum;
          if (shortage > 0) {
            const compensatingPortion = shortage / (settings.columnWidths.length - 1);
            targetWidths = targetWidths.map((w, i) => index === i ? w : w + compensatingPortion);
          }
          return { widths: targetWidths, ...rest };
        });
      });
    },
    [ settings.columns, settings.columnWidths, pageWidth, adjustedColumnWidthSum ],
  );

  const handleResizeStart = useCallback(
    (index) => (e: Event, { x }: DraggableData) => {
      setIsResizing(true);
      const column = settings.columns[index];
      const startWidth = settings.columnWidths[index];
      const minWidth = DEFAULT_COLUMN_WIDTHS[column];
      const deltaX = startWidth - minWidth;
      const minX = x - deltaX;
      setWidthData(({ widths, ...rest }) => ({ minX, widths, ...rest }));
    },
    [ setWidthData, settings.columns, settings.columnWidths ],
  );

  const handleResizeStop = useCallback(() => {
    const newWidths = getUpscaledWidths(widthData.widths);
    setIsResizing(false);
    updateSettings({ columnWidths: newWidths });
  }, [ updateSettings, widthData, getUpscaledWidths ]);

  const onHeaderCell = useCallback(
    (index, columnDefs) => {
      return () => {
        const filterActive = !!columnDefs?.isFiltered?.(settings);
        return {
          columnName: columnDefs.title,
          dragState,
          dropLeftStyle: { ...widthData?.dropLeftStyles?.[index] },
          dropRightStyle: { ...widthData?.dropRightStyles?.[index] },
          filterActive,
          index,
          isResizing,
          moveColumn,
          onResize: handleResize(index),
          onResizeStart: handleResizeStart(index),
          onResizeStop: handleResizeStop,
          width: widthData?.widths[index],
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
      dragState,
      isResizing,
    ],
  );

  const renderColumns: ColumnsType<ExperimentItem> = useMemo(
    () =>
      [
        ...settings.columns.map((columnName, index) => {
          const column = columnDefs[columnName];
          const columnWidth = widthData.widths[index];
          const sortOrder =
          column.key === settings.sortKey ? (settings.sortDesc ? 'descend' : 'ascend') : null;

          return {
            onHeaderCell: onHeaderCell(index, column),
            sortOrder,
            width: columnWidth,
            ...column,
          };
        }),
        { ...columnDefs.action, width: WIDGET_COLUMN_WIDTH },
      ] as ColumnsType<ExperimentItem>,

    [ settings.columns, widthData, settings.sortKey, settings.sortDesc, columnDefs, onHeaderCell ],
  );

  const components = {
    body: {
      cell: Cell,
      row: Row,
    },
    header: { cell: HeaderCell },
  };
  return (
    <div ref={tableRef}>
      <Spinner spinning={spinning}>
        <Table
          bordered
          /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
          columns={renderColumns as ColumnsType<any>}
          components={components}
          dataSource={dataSource}
          tableLayout="fixed"
          onChange={handleChange}
          onRow={(record, index) =>
            ({
              areRowsSelected,
              ContextMenu,
              index,
              record,
            } as React.HTMLAttributes<HTMLElement>)
          }
          {...props}
        />
      </Spinner>
    </div>
  );
};

export default InteractiveTable;
