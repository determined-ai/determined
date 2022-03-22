import { Table } from 'antd';
import { SpinProps } from 'antd/es/spin';
import { TableProps } from 'antd/es/table';
import { ColumnsType, ColumnType, SorterResult } from 'antd/es/table/interface';
import { DEFAULT_COLUMN_WIDTHS, Settings } from 'pages/ExperimentList.settings';
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
import {
  ExperimentItem,
} from 'types';

import css from './InteractiveTable.module.scss';
import Spinner from './Spinner';

const DEFAULT_RESIZE_THROTTLE_TIME = 10;
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

interface HeaderCellProps {
  className: string;
  columnName: string;
  filterActive: boolean;
  index: number;
  moveColumn: (source: number, destination: number) => void;
  onResize: ResizeCallback;
  onResizeStop: ResizeCallback;
  title: unknown;
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

const HeaderCell = ({
  onResize,
  onResizeStop,
  width,
  className,
  columnName,
  filterActive,
  moveColumn,
  index,
  title: unusedTitleFromAntd,
  ...props
}: HeaderCellProps) => {
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
    return <th className={className} {...props} />;
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
          title={columnName}
          {...props}
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
  const tableRef = useRef<HTMLDivElement>(null);
  const [ widths, setWidths ] = useState(settings?.columnWidths);

  const spinning = !!(loading as SpinProps)?.spinning || loading === true;

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
          const column = settings.columns[index];
          const minWidth = DEFAULT_COLUMN_WIDTHS[column] * 0.70;
          const targetWidth = Math.floor(Math.max(size.width, minWidth));
          const newWidths = settings.columnWidths.map((w: number, i: number) =>
            index === i ? targetWidth : w);
          setWidths(newWidths);
        },
      ),
    [ settings.columnWidths, settings.columns ],
  );

  const handleResizeStop = useCallback(
    (index) => (e: React.SyntheticEvent, { size }: ResizeCallbackData) => {
      const column = settings.columns[index];
      const minWidth = DEFAULT_COLUMN_WIDTHS[column] * 0.70;
      const targetWidth = Math.floor(Math.max(size.width, minWidth));
      const newWidths = settings.columnWidths.map((w: number, i: number) =>
        index === i ? targetWidth : w);
      updateSettings({ columnWidths: newWidths });
    },
    [ updateSettings, settings.columns, settings.columnWidths ],
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
    header: { cell: HeaderCell },
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
            // scroll={tableScroll}
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
