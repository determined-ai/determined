// @ts-nocheck
import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import ReactDOM from "react-dom";
import "antd/dist/antd.min.css";
import './DragSortingTable.css';
import { Table } from "antd";
import { Resizable } from "react-resizable";
import ReactDragListView from "react-drag-listview";
import { processApiError } from 'services/utils';
import tableCss from 'components/ResponsiveTable.module.scss';
import useResize from 'hooks/useResize';
import Spinner from './Spinner';
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';


const type = 'DraggableColumn';

const Cell = ({ isCellRightClickable, ...props }) => <td {...props}/>

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
  const classes = [className];
  if (filterActive) classes.push(tableCss.headerFilterOn);

  const ref = useRef();
  const [{ isOver, dropClassName }, drop] = useDrop({
    accept: type,
    collect: (monitor) => {
      const { index: dragIndex } = monitor.getItem() || {};
      if (dragIndex === index) {
        return {};
      }
      return {
        isOver: monitor.isOver(),
        dropClassName: dragIndex > index ? ' drop-over-leftward' : ' drop-over-rightward',
      };
    },
    drop: (item) => {
      moveColumn(item.index, index);
    },
  });
  const [, drag] = useDrag({
    type,
    item: { index },
    collect: (monitor) => ({
      // isDragging: monitor.isDragging(),
    }),
  });
  drop(drag(ref));

  // const sliderRef = useRef()

  // const [{xOffset}, sliderDrag] = useDrag({
  //   type,
  //   item: { index },
  //   collect: (monitor) => ({
  //     xOffset: monitor.getDifferenceFromInitialOffset()?.x
  //   }),
  // });
  // sliderDrag(sliderRef)
  // console.log(xOffset)




  //  console.log(classes.join(' '));

/*   return (
    <th className={isOver ? dropClassName : ''} style={{ cursor: 'move' }}>
      <div ref={ref} className={classes.join(' ')} {...restProps} title={columnName} />
      <Draggable
          axis="x"
          // defaultClassName="DragHandle"
          // defaultClassNameDragging="DragHandleActive"
          onDrag={onResize}
          onStop={onResizeStop}
          position={{ x: 0 }}
          zIndex={999}
        >
          <span className="react-resizable-handle"></span>
        </Draggable>
      
    </th>
  ); */
  

  return (
    <Resizable
      width={width || 100}
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
      onResizeStop={onResizeStop}
      draggableOpts={{ enableUserSelectHack: false }}
    >
      <th className={isOver ? dropClassName : ''} style={{ cursor: 'move' }}>
      
      <div ref={ref} className={classes.join(' ')} {...restProps} title={columnName} />
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
  const [hasScrollBeenEnabled, setHasScrollBeenEnabled] = useState<boolean>(false);
  const spinning = !!(loading as SpinProps)?.spinning || loading === true;
  const [tableScroll, setTableScroll] = useState(scroll);
  const [widths, setWidths] = useState(settings.columnWidths)

  useEffect(() => setWidths(settings.columnWidths), [settings.columnWidths])

  const handleTableChange = useCallback(
    (tablePagination: any, tableFilters: any, tableSorter: any): void => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<unknown>;
      if (
        !columnKey ||
        !settings.columns
          .find((col) => columnSpec[col]?.key === columnKey)
      )
        return;


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
    [settings, updateSettings]
  );



  const moveColumn = useCallback(
    (fromIndex, toIndex) => {
      const reorderedColumns = [...settings.columns];
      const reorderedWidths = [...settings.columnWidths]
      const col = reorderedColumns.splice(fromIndex, 1)[0];
      const width = reorderedWidths.splice(fromIndex, 1)[0];
      reorderedColumns.splice(toIndex, 0, col);
      reorderedWidths.splice(toIndex, 0, width);
      updateSettings({ columns: reorderedColumns, columnWidths: reorderedWidths });
    },
    [settings.columns, settings.columnWidths],
  );





  const handleResize = useCallback(
    (index) => (e, { size }) => {
      const targetWidth = Math.floor(Math.max(size.width, 80));
      const newWidths = widths.map((w, i) => (index === i ? targetWidth : w));
      setWidths(newWidths);
    },
    [updateSettings, settings.columnWidths]
  );

  const handleResizeStop = useCallback(
    (index) => (e, { size }) => {
      const targetWidth = Math.floor(Math.max(size.width, 80));
      const newWidths = widths.map((w, i) => (index === i ? targetWidth : w));
      setWidths(newWidths);
      updateSettings({ columnWidths: newWidths });
    },
    [updateSettings, settings.columnWidths]
  );

  const onHeaderCell = (index, columnSpec) => {
    return (column) => {
      const filterActive = !!columnSpec?.isFiltered?.(settings);
      return {
        onResize: handleResize(index),
        onResizeStop: handleResizeStop(index),
        moveColumn,
        columnName: columnSpec.title,
        filterActive,
        width: column.width,
        index
      };
    };
  };

  // const widthColumnCount = columns.filter(({ width }) => !width).length;
  // const mergedColumns = columns.map((column) => {
  //   if (column.width) {
  //     return column;
  //   }

  //   return { ...column, width: Math.floor(tableWidth / widthColumnCount) };
  // });




  const renderColumns = useMemo(
    () =>
      settings.columns.map((columnName, index) => {
        const column = columnSpec[columnName];
        // const columnWidth = settings.columnWidths?.[index] ?? 100;
        const columnWidth = widths[index];
        const sortOrder =
          column.key === settings.sortKey ? (settings.sortDesc ? 'descend' : 'ascend') : null;

        return {
          sortOrder,
          width: columnWidth,
          onHeaderCell: onHeaderCell(index, column),
          ...column,
        };
      }).concat(columnSpec.action),
    [settings.columns, widths, settings.sortKey, settings.sortDesc, columnSpec]
  );


  const components = {
    header: {
      cell: ResizableTitle,
    },
    body: {
      cell: Cell,
    },
  };
  return (
    <div ref={tableRef}>
      <Spinner spinning={spinning}>
        {/* <ReactDragListView.DragColumn {...dragProps}> */}
        <DndProvider backend={HTML5Backend}>
          <Table
            bordered
            // onHeaderRow
            components={components}
            columns={renderColumns}
            dataSource={dataSource}
            scroll={tableScroll}
            tableLayout="fixed"
            onChange={handleTableChange}
            {...props}
          />
          </DndProvider>
        {/* </ReactDragListView.DragColumn> */}
      </Spinner>
    </div>
  );
};


export default ResponsiveTable