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


const ResizableTitle = ({ onResize, width, className, columnName, filterActive, style,...restProps }) => {

  if (!columnName) return <th className={`${className} notColumn`} {...restProps} />;
  
  const fullClassName = filterActive ? `${className} ${tableCss.headerFilterOn}` : className
  return (
    <Resizable
      width={width || 5}
      // height={0}
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

      <th style={{ cursor: "move", ...style }}>
        <div  className={fullClassName} {...restProps} title={columnName} />
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
  // const [columns, setColumns] = useState(visibleColumns);

  const tableRef = useRef<HTMLDivElement>(null);
  const [hasScrollBeenEnabled, setHasScrollBeenEnabled] = useState<boolean>(false);
  const resize = useResize(tableRef);
  const spinning = !!(loading as SpinProps)?.spinning || loading === true;
  const [tableScroll, setTableScroll] = useState(scroll);

  const handleTableChange = useCallback(
    (tablePagination: any, tableFilters: any, tableSorter: any): void => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<unknown>;
      if (!columnKey || !settings.columns?.find((column) => column.key === columnKey)) return;

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

  // useEffect(() => {
  //   console.log("columns... changed?")
  //   setColumnOrder(columns);
  // }, [columns]);

  const dragProps = {
    onDragEnd: (fromIndex, toIndex) => {
      const reorderedColumns = [...settings.columns];
      const reorderedWidths = [...settings.columnWidths]
      const col = reorderedColumns.splice(fromIndex, 1)[0];
      const width = reorderedWidths.splice(fromIndex, 1)[0];
      // console.log({ fromIndex, toIndex });
      reorderedColumns.splice(toIndex, 0, col);
      reorderedWidths.splice(toIndex, 0, width);
      updateSettings({ columns: reorderedColumns, widths: reorderedWidths });
    },
    nodeSelector: 'th',
    handleSelector: '.ant-table-cell',
    ignoreSelector: '.notColumn',
  };

  const components = {
    header: {
      cell: ResizableTitle,
    },
  };

  const handleResize = useCallback(
    (index) => (e, { size }) => {
      // const newWidth = Math.max(size.width, 100)
      const newWidth = size.width
      // const ostensibleTarget = columns[index];
      // const resizedColumns = columns.map((col, i) =>
      //   index === i ? { ...col, width: width } : col
      // );
      const newWidths = settings.columnWidths.map((w, i) => index === i ? newWidth : w)
      console.log({newWidths})
      // const colWidths = resizedColumns.map((c) => `${c.title}: ${c.width}`).join(' // ');
      // console.log({index,width, ostensibleTarget, colWidths })
      updateSettings({ columnWidths: newWidths });
    },
    [updateSettings]
  );

  const onHeaderCell = (index, columnSpec) => {
    return (column) => {
      const filterActive = !!columnSpec?.isFiltered?.(settings);
      return {
        onResize: handleResize(index),
        columnName: columnSpec.title,
        filterActive,
        ...column,
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
        // console.log(settings.columnWidths.join(" "))
        const columnWidth = settings.columnWidths?.[index]  ?? 100;
        // if (columnName === "description")
        // console.log(`columnwidehthte ${columnWidth} ${index}`)
        const sortOrder =
          column.key === settings.sortKey ? (settings.sortDesc ? 'descend' : 'ascend') : null;

        return {
          sortOrder,
          width: columnWidth,
          onHeaderCell: onHeaderCell(index, column),
          ...column,
        };
      }),
    [settings.columns, settings.columnWidth, columnSpec]
  );

  // console.log(renderColumns.map(x=>x.key))
  // console.log(renderColumns.map((x) => `${x.title} ${x.width}`));
  // console.log(settings)
  return (
    <div ref={tableRef}>
      <Spinner spinning={spinning}>
        <ReactDragListView.DragColumn {...dragProps}>
          <Table
            bordered
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