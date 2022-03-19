// @ts-nocheck
import { Table } from 'antd';
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import 'antd/dist/antd.min.css';
import './ResponsiveTable.css';
import { Resizable } from 'react-resizable';

import tableCss from 'components/ResponsiveTable.module.scss';
import useResize from 'hooks/useResize';

import Spinner from './Spinner';

import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';

const type = 'DraggableColumn';

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
      <div className={tableCss.rightClickableCellWrapper} {...rightClickableCellProps}>
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
}) => {
  if (!columnName) {
    return <th className={`${className} notColumn`} {...restProps} />;
  }
  const classes = [ className ];
  if (filterActive) classes.push(tableCss.headerFilterOn);

  const ref = useRef();
  const [ { isOver, dropClassName }, drop ] = useDrop({
    accept: type,
    collect: (monitor) => {
      const { index: dragIndex } = monitor.getItem() || {};
      if (dragIndex === index) {
        return {};
      }
      return {
        dropClassName: dragIndex > index ? ' drop-over-leftward' : ' drop-over-rightward',
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

  return (
    <Resizable
      draggableOpts={{ enableUserSelectHack: false }}
      handle={(
        <span
          className="react-resizable-handle"
          onClick={(e) => {
            e.stopPropagation();
          }}
        />
      )}
      height={0}
      width={width || 100}
      onResize={onResize}
      onResizeStop={onResizeStop}>
      <th className={isOver ? dropClassName : ''}>
        <div
          className={classes.join(' ')}
          ref={ref}
          style={{ cursor: 'move', marginLeft: 4, marginRight: 12 }}
          {...restProps}
          title={columnName}
        />
      </th>
    </Resizable>
  );
};

const ResponsiveTable = ({
  dataSource,
  columnSpec,
  settings,
  updateSettings,
  areRowsRightClickable,
  ContextMenu,
  areRowsSelected,
  loading,
  ...props
}) => {
  const tableRef = useRef<HTMLDivElement>(null);
  const [ hasScrollBeenEnabled, setHasScrollBeenEnabled ] = useState<boolean>(false);
  const spinning = !!(loading as SpinProps)?.spinning || loading === true;
  const [ tableScroll, setTableScroll ] = useState(scroll);
  const [ widths, setWidths ] = useState(settings.columnWidths);
  const resize = useResize(tableRef);

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

  const handleTableChange = useCallback(
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
            onChange={handleTableChange}
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
