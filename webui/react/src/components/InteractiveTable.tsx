// @ts-nocheck

import { Table } from 'antd';
import { SpinProps } from 'antd/es/spin';
import { TableProps } from 'antd/es/table';
import { SorterResult } from 'antd/es/table/interface';
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import 'antd/dist/antd.min.css';
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Resizable } from 'react-resizable';

import useResize from 'hooks/useResize';

import css from './ResponsiveTable.bak.module.scss';
import Spinner from './Spinner';

const type = 'DraggableColumn';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
type Comparable = any;

interface Settings {
  sortDesc: boolean;
  sortKey: Comparable;
  tableLimit: number;
  tableOffset: number;
}

interface ContextMenuProps {
  onVisibleChange: (visible: boolean) => void;
  record: Record<string, unknown>;
}

interface ResponsiveTableProps<RecordType> extends TableProps<RecordType> {
  ContextMenu?: React.FC<ContextMenuProps>;
  areRowsRightClickable?: boolean;
  areRowsSelected?: boolean;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
type ResponsiveTable = <T extends object>(props: ResponsiveTableProps<T>) => JSX.Element;

interface RowProps {
  ContextMenu: React.FC<ContextMenuProps>;
  areRowsSelected?: boolean;
  children?: React.ReactNode;
  className?: string;
  record: Record<string, unknown>;
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

export const handleTableChange = (
  columns: {key?: Comparable}[],
  settings: Settings,
  updateSettings: (s: Settings, b: boolean) => void,
) => {
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  return (tablePagination: any, tableFilters: any, tableSorter: any): void => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<unknown>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    const newSettings = {
      sortDesc: order === 'descend',
      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      sortKey: columnKey as any,
      tableLimit: tablePagination.pageSize,
      tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
    };
    const shouldPush = settings.tableOffset !== newSettings.tableOffset;
    updateSettings(newSettings, shouldPush);
  };
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
}) => {
  if (!columnName) {
    return <th className={className} {...restProps} />;
  }
  const classes = [ className, css.headerCell ];

  const ref = useRef();
  const [ { isOver, dropClassName }, drop ] = useDrop({
    accept: type,
    collect: (monitor) => {
      const { index: dragIndex } = monitor.getItem() || {};
      if (dragIndex === index) {
        return {};
      }
      return {
        dropClassName: dragIndex > index ? css.dropOverLeftward : css.dropOverRightward,
        isOver: monitor.isOver(),
      };
    },
    drop: (item) => {
      moveColumn(item.index, index);
    },
  });
  const [ , drag ] = useDrag({
    item: { index },
    type,
    // collect: (monitor) => ({
    //   isDragging: monitor.isDragging(),
    // }),
  });
  drop(drag(ref));

  // if (isOver) classes.push(dropClassName)
  classes.push(css.dropOverRightward)

  console.log(classes.join(' '))
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
      width={width || 100}
      onResize={onResize}
      onResizeStop={onResizeStop}>
      <th
        // className={classes.join(' ')}
        // className={isOver ? dropClassName : ''}
        className={`${className} ${css.headerCell}`}
      >
        <div
          // className={css.headerCell}
          className={filterActive ? css.headerFilterOn : ''}
          // className={classes.join(' ')}
          ref={ref}
          style={{ cursor: 'move', marginLeft: 4, marginRight: 12 }}
          {...restProps}
          title={columnName}
        />
      </th>
    </Resizable>
  );
};

const ResponsiveTable: ResponsiveTable = ({
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
    [ settings, updateSettings ],
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
    [ settings.columns, settings.columnWidths ],
  );

  const handleResize = useCallback(
    (index) => (e, { size }) => {
      const targetWidth = Math.floor(Math.max(size.width, 30));
      const newWidths = widths.map((w, i) => (index === i ? targetWidth : w));
      setWidths(newWidths);
    },
    [ updateSettings, settings.columnWidths ],
  );

  const handleResizeStop = useCallback(
    (index) => (e, { size }) => {
      const targetWidth = Math.floor(Math.max(size.width, 30));
      const newWidths = widths.map((w, i) => (index === i ? targetWidth : w));
      setWidths(newWidths);
      updateSettings({ columnWidths: newWidths });
    },
    [ updateSettings, settings.columnWidths ],
  );

  const onHeaderCell = (index, columnSpec) => {
    return (column) => {
      const filterActive = !!columnSpec?.isFiltered?.(settings);
      return {
        columnName: columnSpec.title,
        filterActive,
        index,
        moveColumn,
        onResize: handleResize(index),
        onResizeStop: handleResizeStop(index),
        width: column.width,
      };
    };
  };

  const renderColumns = useMemo(
    () =>
      settings.columns.map((columnName, index) => {
        const column = columnSpec[columnName];
        // const columnWidth = settings.columnWidths?.[index] ?? 100;
        const columnWidth = widths[index];
        const sortOrder =
          column.key === settings.sortKey ? (settings.sortDesc ? 'descend' : 'ascend') : null;

        return {
          onHeaderCell: onHeaderCell(index, column),
          sortOrder,
          width: columnWidth,
          ...column,
        };
      }).concat(columnSpec.action),
    [ settings.columns, widths, settings.sortKey, settings.sortDesc, columnSpec ],
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
            columns={renderColumns}
            components={components}
            dataSource={dataSource}
            scroll={tableScroll}
            // tableLayout="fixed"
            tableLayout="auto"
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
        </DndProvider>
      </Spinner>
    </div>
  );
};

export default ResponsiveTable;
