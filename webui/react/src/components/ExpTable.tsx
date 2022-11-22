import { Column, ColumnOrderState, flexRender, Table } from '@tanstack/react-table';
import { Pagination } from 'antd';
import React, { useCallback } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import {
  // ColumnDef,
  InteractiveTableSettings,
  Row,
} from 'components/Table/InteractiveTable';
import { UpdateSettings } from 'hooks/useSettings';
import { ExperimentColumnName } from 'pages/ExperimentList.settings';
import { ExperimentItem } from 'types';

import css from './ExpTable.module.scss';

export interface ContextMenuProps<RecordType> {
  children: React.ReactNode;
  onVisibleChange: (visible: boolean) => void;
  record: RecordType;
}

interface ExpTableProps {
  ContextMenu?: React.FC<ContextMenuProps<ExperimentItem>>;
  areRowsSelected?: boolean;
  interactiveColumns?: boolean;
  loading?: boolean;
  numOfPinned?: number;
  settings: InteractiveTableSettings;
  table: Table<ExperimentItem>;
  total: number;
  updateSettings: UpdateSettings<InteractiveTableSettings>;
}

const ExpTable: React.FC<ExpTableProps> = ({
  settings,
  updateSettings,
  ContextMenu,
  table,
  total,
}: ExpTableProps) => {
  const handleSort = useCallback(
    (header) => {
      const sortKey = header.id;
      let sortDesc = settings.sortDesc;
      if (settings.sortKey === sortKey) sortDesc = !sortDesc;
      updateSettings({
        sortDesc,
        sortKey,
      });
    },
    [updateSettings, settings],
  );

  const updatePagination = useCallback(
    (page, pageSize) => {
      updateSettings({
        tableLimit: pageSize,
        tableOffset: (page - 1) * pageSize,
      });
    },
    [updateSettings],
  );

  const DraggableColumnHeader: React.FC<{
    header;
    table;
  }> = ({ header, table }) => {
    const reorderColumn = (draggedColumnId: string, targetColumnId: string): ColumnOrderState => {
      const columnOrder = table.getState().columnOrder;
      columnOrder.splice(
        columnOrder.indexOf(targetColumnId),
        0,
        columnOrder.splice(columnOrder.indexOf(draggedColumnId), 1)[0] as string,
      );
      return [...columnOrder];
    };
    const { column } = header;

    const [, dropRef] = useDrop({
      accept: 'column',
      drop: (draggedColumn: Column<ExperimentItem>) => {
        const newColumnOrder = reorderColumn(draggedColumn.id, column.id);
        table.setColumnOrder(newColumnOrder);
        updateSettings({
          columns: newColumnOrder as ExperimentColumnName[],
        });
      },
    });

    const [{ isDragging }, dragRef, previewRef] = useDrag({
      collect: (monitor) => ({
        isDragging: monitor.isDragging(),
      }),
      item: () => column,
      type: 'column',
    });

    return (
      <th
        className="ant-table-cell"
        colSpan={header.colSpan}
        key={header.id}
        ref={dropRef}
        style={{
          opacity: isDragging ? 0.5 : 1,
          width: header.getSize(),
        }}>
        <div
          ref={previewRef}
          {...{
            className: header.column.getCanSort() ? 'cursor-pointer select-none' : '',
            onClick: () => {
              if (header.column.getCanSort()) {
                handleSort(header);
              }
            },
          }}>
          <div ref={dragRef}>
            {header.isPlaceholder
              ? null
              : flexRender(header.column.columnDef.header, header.getContext())}
            {settings.sortKey === header.id ? (settings.sortDesc ? ' 🔽' : ' 🔼') : ''}
          </div>
          <div
            {...{
              className: `${header.column.getCanResize() ? css.resizer : ''} ${header.column.getIsResizing() ? css.isResizing : ''}`,
              onMouseDown: header.getResizeHandler(),
              onTouchStart: header.getResizeHandler(),
              style: {
                transform: '',
              },
            }}
          />
        </div>
      </th>
    );
  };

  return (
    <div className="tableContainer">
      <table
        {...{
          style: {
            width: table.getCenterTotalSize(),
          },
        }}>
        <thead className="ant-table-thead">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <DraggableColumnHeader header={header} key={header.id} table={table} />
              ))}
            </tr>
          ))}
        </thead>
        <tbody className="ant-table-tbody">
          {table.getRowModel().rows.map((row) => (
            <Row
              className="ant-table-row ant-table-row-level-0"
              ContextMenu={ContextMenu}
              index={parseInt(row.id)}
              key={row.id}
              record={row.original}>
              {row.getVisibleCells().map((cell) => (
                <td
                  className="ant-table-cell"
                  key={cell.id}
                  style={{
                    height: 60,
                    width: cell.column.getSize(),
                    overflow: 'hidden',
                    paddingBottom: 0,
                    paddingTop: 0,
                  }}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </Row>
          ))}
        </tbody>
      </table>
      <Pagination
        defaultCurrent={settings.tableOffset + 1}
        showSizeChanger
        total={total}
        onChange={updatePagination}
        onShowSizeChange={updatePagination}
      />
    </div>
  );
};
export default ExpTable;
