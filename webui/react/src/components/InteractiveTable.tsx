import { Table } from 'antd';
import { SpinProps } from 'antd/es/spin';
import { TableProps } from 'antd/es/table';
import { ColumnsType, ColumnType, SorterResult } from 'antd/es/table/interface';
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import 'antd/dist/antd.min.css';
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Resizable, ResizeCallbackData } from 'react-resizable';
import { throttle } from 'throttle-debounce';

import useResize from 'hooks/useResize';
import { Settings } from 'pages/ExperimentList.settings';
import {
  ExperimentItem,
} from 'types';

import css from './InteractiveTable.module.scss';
import Spinner from './Spinner';

const DEFAULT_RESIZE_THROTTLE_TIME = 20;
const MIN_COLUMN_WIDTH = 40;

const type = 'DraggableColumn';

type ResizeCallback = ((e: React.SyntheticEvent, data: ResizeCallbackData) => void) | undefined;

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
  areRowsRightClickable?: boolean;
  areRowsSelected?: boolean;
  columnSpec: ColumnDefs<string, RecordType>;
  settings: Settings;
  updateSettings: (settings: Partial<Settings>, shouldPush?: boolean) => void;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
type InteractiveTable = <T extends object>(props: InteractiveTableProps<T>) => JSX.Element;

interface RowProps {
  ContextMenu: React.FC<ContextMenuProps>;
  areRowsSelected?: boolean;
  children?: React.ReactNode;
  className?: string;
  record: Record<string, unknown>;
}

interface ResizableTitleProps {
  className: string;
  columnName: string;
  filterActive: boolean;
  index: number;
  moveColumn: (source: number, destination: number) => void;
  onResize: ResizeCallback;
  onResizeStop: ResizeCallback;
  width: number;
}

interface CellProps {
  children?: React.ReactNode;
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

  const [ rowHovered, setRowHovered ] = useState(false);
  const [ rightClickableCellHovered, setRightClickableCellHovered ] = useState(false);
  const [ contextMenuOpened, setContextMenuOpened ] = useState(false);

  if (areRowsSelected) {
    return <tr className={className} {...props}>{children}</tr>;
  }

  const rightClickableCellProps = {
    onContextMenu: (e : React.MouseEvent) => e.stopPropagation(),
    onMouseEnter: () => setRightClickableCellHovered(true),
    onMouseLeave: () => setRightClickableCellHovered(false),
  };

  const rowContextMenuTriggerableOrOpen =
    (rowHovered && !rightClickableCellHovered) || contextMenuOpened;

  return record ? (
    <RightClickableRowContext.Provider value={{ ...rightClickableCellProps }}>
      <ContextMenu record={record} onVisibleChange={setContextMenuOpened}>
        <tr
          className={
            rowContextMenuTriggerableOrOpen ? `${className} ant-table-row-selected` : className
          }
          onMouseEnter={() => setRowHovered(true)}
          onMouseLeave={() => setRowHovered(false)}
          {...props}>
          {children}
        </tr>
      </ContextMenu>
    </RightClickableRowContext.Provider>
  ) : (
    <tr className={className} {...props}>{children}</tr>
  );
};

const Cell = ({ children, isCellRightClickable, ...props }: CellProps) => {
  const rightClickableCellProps = useContext(RightClickableRowContext);
  if (!isCellRightClickable) return <td {...props}>{children}</td>;
  return (
    <td {...props}>
      <div className={css.rightClickableCellWrapper} {...rightClickableCellProps}>
        {children}
      </div>
    </td>
  );
};

const ResizableTitle = ({
  onResize,
  onResizeStop,
  width,
  className,
  columnName,
  filterActive,
  moveColumn,
  index,
  ...restProps
}: ResizableTitleProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const classes = [ css.headerCell ];

  const [ { isOver, dropClassName }, drop ] = useDrop({
    accept: type,
    collect: (monitor) => {

      const dragItem = (monitor.getItem() || {}) as DndItem;
      const dragIndex = dragItem?.index;
      if (dragIndex == null || dragIndex === index) {
        return {};
      }
      return {
        dropClassName: dragIndex > index ? css.dropOverLeftward : css.dropOverRightward,
        isOver: monitor.isOver(),
      };
    },
    drop: (item: DndItem) => {
      if (item.index != null) {
        moveColumn(item.index, index);
      }
    },
  });
  const [ , drag ] = useDrag({
    item: { index },
    type,
  });
  drop(drag(ref));

  if (isOver) classes.push(dropClassName ?? '');
  if (filterActive) classes.push(css.headerFilterOn);

  if (!columnName) {
    return <th className={className} {...restProps} />;
  }
  return (
    <Resizable
      draggableOpts={{ enableUserSelectHack: false }}
      handle={(
        <span
          className={css.columnResizeHandle}
          onClick={(e) => {
            e.stopPropagation();
          }}
        />
      )}
      height={0}
      width={width || MIN_COLUMN_WIDTH}
      onResize={onResize}
      onResizeStop={onResizeStop}>
      <th
        className={classes.join(' ')}>
        <div
          className={className}
          ref={ref}
          {...restProps}
          title={columnName}
        />
      </th>
    </Resizable>
  );
};

const InteractiveTable: InteractiveTable = ({
  loading,
  scroll,
  dataSource,
  columnSpec,
  settings,
  updateSettings,
  areRowsRightClickable,
  ContextMenu,
  areRowsSelected,
  ...props
}) => {
  const [ hasScrollBeenEnabled, setHasScrollBeenEnabled ] = useState<boolean>(false);
  const [ tableScroll, setTableScroll ] = useState(scroll);
  const tableRef = useRef<HTMLDivElement>(null);
  const [ widths, setWidths ] = useState(settings?.columnWidths);
  const resize = useResize(tableRef);

  const spinning = !!(loading as SpinProps)?.spinning || loading === true;

  useEffect(() => {
    if (!tableRef.current || resize.width === 0) return;

    const tables = tableRef.current.getElementsByTagName('table');
    if (tables.length === 0) return;

    const rect = tables[0].getBoundingClientRect();

    /*
     * ant table scrolling has an odd behaviour. If scroll.x is set to 'max-content' initially
     * it will show the scroll bar. We need to set it to undefined the first time if scrolling
     * is not needed, and 'max-content' if we want to disable scrolling after it has been displayed.
     */
    let scrollX: 'max-content'|undefined|number = (
      hasScrollBeenEnabled ? 'max-content' : undefined
    );
    if (rect.width > resize.width) {
      scrollX = rect.width;
      setHasScrollBeenEnabled(true);
    }

    setTableScroll({
      x: scrollX,
      y: scroll?.y,
    });
  }, [ hasScrollBeenEnabled, resize, scroll ]);

  useEffect(() => setWidths(settings.columnWidths), [ settings.columnWidths ]);

  const handleChange = useCallback(
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    (tablePagination: any, tableFilters: any, tableSorter: any): void => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<unknown>;
      if (!columnKey || !settings.columns.find((col) => columnSpec[col]?.key === columnKey)) return;

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
    [ settings, updateSettings, columnSpec ],
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
    (index) =>
      throttle(
        DEFAULT_RESIZE_THROTTLE_TIME,
        (e: React.SyntheticEvent, { size }: ResizeCallbackData) => {
          const targetWidth = Math.floor(Math.max(size.width, MIN_COLUMN_WIDTH));
          const newWidths = widths.map((w: number, i : number) => (index === i ? targetWidth : w));
          setWidths(newWidths);
        },
      ),
    [ widths ],
  );

  const handleResizeStop = useCallback(
    (index) => (e: React.SyntheticEvent, { size }: ResizeCallbackData) => {
      const targetWidth = Math.floor(Math.max(size.width, MIN_COLUMN_WIDTH));
      const newWidths = widths.map((w: number, i: number) => (index === i ? targetWidth : w));
      // setWidths(newWidths);
      updateSettings({ columnWidths: newWidths });
    },
    [ updateSettings, widths ],
  );

  const onHeaderCell = useCallback((index, columnSpec) => {
    return () => {
      const filterActive = !!columnSpec?.isFiltered?.(settings);
      return {
        columnName: columnSpec.title,
        filterActive,
        index,
        moveColumn,
        onResize: handleResize(index),
        onResizeStop: handleResizeStop(index),
        width: widths[index],
      };
    };
  }, [ handleResize, handleResizeStop, widths, moveColumn, settings ]);

  const renderColumns: ColumnsType<ExperimentItem> = useMemo(
    () => [
      ...settings.columns.map((columnName, index) => {
        const column = columnSpec[columnName];
        const columnWidth = widths[index];
        const sortOrder =
          column.key === settings.sortKey ? (settings.sortDesc ? 'descend' : 'ascend') : null;

        return {
          onHeaderCell: onHeaderCell(index, column),
          sortOrder,
          width: columnWidth,
          ...column,
        };
      }, columnSpec.action) as ColumnsType<ExperimentItem>,
    ],
    [ settings.columns, widths, settings.sortKey, settings.sortDesc, columnSpec, onHeaderCell ],
  );

  const components = {
    body: {
      cell: Cell,
      row: Row,
    },
    header: { cell: ResizableTitle },
  };
  return (
    <div ref={tableRef}>
      <Spinner spinning={spinning}>
        <DndProvider backend={HTML5Backend}>
          <Table
            bordered
            /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
            columns={renderColumns as ColumnsType<any>}
            components={components}
            dataSource={dataSource}
            scroll={tableScroll}
            tableLayout="fixed"
            onChange={handleChange}
            onRow={(record, index) => ({
              areRowsSelected,
              ContextMenu,
              index,
              record,
            } as React.HTMLAttributes<HTMLElement>)}
            {...props}
          />
        </DndProvider>
      </Spinner>
    </div>
  );
};

export default InteractiveTable;
