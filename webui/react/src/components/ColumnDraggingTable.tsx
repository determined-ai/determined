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

const Cell = ({ isCellRightClickable, ...props }) => <td {...props}/>

const ResizableTitle = ({
  onResize,
  onResizeStop,
  width,
  className,
  columnName,
  filterActive,
  ...restProps
}) => {
  if (!columnName) {
    //   console.log(restProps)
    return <th className={`${className} notColumn`} {...restProps} />;
  }

  const fullClassName = filterActive ? `${className} ${tableCss.headerFilterOn}` : className;
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
      onResizeStop={onResizeStop}
      draggableOpts={{ enableUserSelectHack: false }}
    >
      <th style={{ cursor: 'move' }}>
        <div className={fullClassName} {...restProps} title={columnName} />
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
  const resize = useResize(tableRef);
  const spinning = !!(loading as SpinProps)?.spinning || loading === true;
  const [tableScroll, setTableScroll] = useState(scroll);
  const [widths, setWidths] = useState(settings.columnWidths)

  useEffect(() => setWidths(settings.columnWidths), [settings.columnWidths])

  const handleTableChange = useCallback(
    (tablePagination: any, tableFilters: any, tableSorter: any): void => {
      console.log("table change")
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


  const dragProps = {
    onDragEnd: (fromIndex, toIndex) => {
      const reorderedColumns = [...settings.columns];
      const reorderedWidths = [...settings.columnWidths]
      const col = reorderedColumns.splice(fromIndex, 1)[0];
      const width = reorderedWidths.splice(fromIndex, 1)[0];
      reorderedColumns.splice(toIndex, 0, col);
      reorderedWidths.splice(toIndex, 0, width);
      updateSettings({ columns: reorderedColumns, columnWidths: reorderedWidths });
    },
    nodeSelector: 'th',
    handleSelector: '.ant-table-cell',
    ignoreSelector: '.notColumn',
  };

  const components = {
    header: {
      cell: ResizableTitle,
    },
    body: {
      cell: Cell
    }
  };

  // console.log(widths[0])

  const handleResize = useCallback(
    (index) => (e, { size }) => {
      const targetWidth = Math.floor(Math.max(size.width, 80));
      
      // if (targetWidth !== settings.columnWidths[index]) {
      const newWidths = widths.map((w, i) => (index === i ? targetWidth : w));
      setWidths(newWidths);
      // updateSettings({ columnWidths: newWidths });
      // }
    },
    [updateSettings]
  );

  const handleResizeStop = useCallback(
    (index) => (e, { size }) => {
      console.log("resize stop")
      const targetWidth = Math.floor(Math.max(size.width, 80))
      // if (targetWidth !== settings.columnWidths[index]) {
        const newWidths = widths.map((w, i) => (index === i ? targetWidth : w));
        setWidths(newWidths);
        updateSettings({ columnWidths: newWidths });
      // }
    },
    [updateSettings]
  );

  const onHeaderCell = (index, columnSpec) => {
    return (column) => {
      const filterActive = !!columnSpec?.isFiltered?.(settings);
      return {
        onResize: handleResize(index),
        onResizeStop: handleResizeStop(index),
        columnName: columnSpec.title,
        filterActive,
        width: column.width
      };
    };
  };

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
    let scrollX: 'max-content' | undefined | number = hasScrollBeenEnabled
      ? 'max-content'
      : undefined;
    if (rect.width > resize.width) {
      scrollX = rect.width;
      setHasScrollBeenEnabled(true);
    }

    setTableScroll({
      x: scrollX,
      y: scroll?.y,
    });
  }, [hasScrollBeenEnabled, resize, scroll]);

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

    // console.log(renderColumns.map((x) => `${x.width}`));
    // console.log(renderColumns.map((x) => `${x.title}`));

  return (
    <div ref={tableRef}>
      <Spinner spinning={spinning}>
        <ReactDragListView.DragColumn {...dragProps}>
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
        </ReactDragListView.DragColumn>
      </Spinner>
    </div>
  );
};


export default ResponsiveTable