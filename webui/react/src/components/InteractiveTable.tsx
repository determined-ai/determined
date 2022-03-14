// @ts-nocheck
import { Table } from 'antd';
import { SpinProps } from 'antd/es/spin';
import { TableProps } from 'antd/es/table';
import { SorterResult } from 'antd/es/table/interface';
import React, { createContext, useContext, useEffect, useRef, useState, useCallback } from 'react';
import ReactDOM from 'react-dom';
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Resizable } from "react-resizable";
import ReactDragListView from "react-drag-listview";
// import update from 'immutability-helper';

import useResize from 'hooks/useResize';

import css from './ResponsiveTable.module.scss';
import Spinner from './Spinner';


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

const ResizableTitle = ({ onResize, width, ...restProps }) => {

  if (!width) {
    return <th {...restProps} />;
  }

  return (
    <Resizable
      width={width}
      height={0}
      handle={
        <span
          className="react-resizable-handle"
          onClick={(e) => {
            e.stopPropagation();
          }}
        />
      }
      onResize={onResize}
      draggableOpts={{ enableUserSelectHack: false }}
    >
      <th {...restProps} />
    </Resizable>
  );
};

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

  const ref = useRef();
  const [{ isOver, dropClassName }, drop] = useDrop({
    accept: type,
    collect: monitor => {
      const { index: dragIndex } = monitor.getItem() || {};
      if (dragIndex === index) {
        return {};
      }
      return {
        isOver: monitor.isOver(),
        dropClassName: dragIndex < index ? ' drop-over-downward' : ' drop-over-upward',
      };
    },
    drop: item => {
      moveRow(item.index, index);
    },
  });
  const [, drag] = useDrag({
    type,
    item: { index },
    collect: monitor => ({
      isDragging: monitor.isDragging(),
    }),
  });
  drop(drag(ref));

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
          ref={ref}
          // className={`${className}${isOver ? dropClassName : ''}`}
          // style={{ cursor: 'move'}}
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

const type = 'DraggableBodyRow';

const DraggableBodyRow = ({ index, moveRow, className, style, ...restProps }) => {
  const ref = useRef();
  const [{ isOver, dropClassName }, drop] = useDrop({
    accept: type,
    collect: monitor => {
      const { index: dragIndex } = monitor.getItem() || {};
      if (dragIndex === index) {
        return {};
      }
      return {
        isOver: monitor.isOver(),
        dropClassName: dragIndex < index ? ' drop-over-downward' : ' drop-over-upward',
      };
    },
    drop: item => {
      moveRow(item.index, index);
    },
  });
  const [, drag] = useDrag({
    type,
    item: { index },
    collect: monitor => ({
      isDragging: monitor.isDragging(),
    }),
  });
  drop(drag(ref));

  return (
    <tr
      ref={ref}
      className={`${className}${isOver ? dropClassName : ''}`}
      style={{ cursor: 'move', ...style }}
      {...restProps}
    />
  );
};


const ResponsiveTable: ResponsiveTable = ({
  loading,
  scroll,
  areRowsRightClickable,
  ContextMenu,
  areRowsSelected,
  dataSource,
  columns,
  ...props
}) => {
  const [ hasScrollBeenEnabled, setHasScrollBeenEnabled ] = useState<boolean>(false);
  const [ tableScroll, setTableScroll ] = useState(scroll);
  const tableRef = useRef<HTMLDivElement>(null);
  const resize = useResize(tableRef);
  

  const [data, setData] = useState(dataSource)
  const [columnOrder, setColumnOrder] = useState(columns)

  useEffect(() => {
    setData(dataSource)
  }, [dataSource])

  useEffect(() => {
    setColumnOrder(columns)
  }, [columns])

  const dragProps = {
    onDragEnd: (fromIndex, toIndex) => {
      console.log({fromIndex, toIndex})
      const columns = [...columnOrder];


      const item = columns.splice(fromIndex, 1)[0];
      columns.splice(toIndex, 0, item);
      setColumns(columns)
    },
    nodeSelector: 'th',
    handleSelector: '.dragHandler',
    ignoreSelector: 'react-resizable-handle',
  };

  // handleResize = (index) => (e, { size }) => {
  //   this.setState(({ columns }) => {
  //     const nextColumns = [...columns];
  //     nextColumns[index] = {
  //       ...nextColumns[index],
  //       width: size.width
  //     };
  //     return { columns: nextColumns };
  //   });
  // };

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

  const moveRow = useCallback(
    (dragIndex, hoverIndex) => {
      const dragRow = data?.[dragIndex];
      console.log({dragRow})
      dragRow &&
        setData(
          update(data, {
            $splice: [
              [dragIndex, 1],
              [hoverIndex, 0, dragRow],
            ],
          })
        );
    },
    [data],
  );

  const col = columnOrder.map((col, index) => ({
    ...col,
    onHeaderCell: (column) => ({
      width: column.width,
      // onResize: this.handleResize(index)
    })
  }));

  return (
    <div ref={tableRef}>
      <Spinner spinning={spinning}>
        <ReactDragListView.DragColumn {...dragProps}>
          <Table
            columns={col}
            components={
              areRowsRightClickable
                ? {
                    header: {
                      cell: ResizableTitle,
                    },
                    body: {
                      cell: Cell,
                      //   row: DraggableBodyRow,
                    },
                  }
                : undefined
            }
            scroll={tableScroll}
            tableLayout="auto"
            onRow={(record, index) =>
              ({
                // areRowsSelected,
                // ContextMenu,
                // record,
                // index,
                // moveRow,
              } as React.HTMLAttributes<HTMLElement>)
            }
            dataSource={data}
            {...props}
          />
        </ReactDragListView.DragColumn>
      </Spinner>
    </div>
  );
};

export default ResponsiveTable;
