import DataEditor, {
  CellClickedEventArgs,
  CompactSelection,
  CustomCell,
  DataEditorProps,
  DataEditorRef,
  GridCell,
  GridCellKind,
  GridColumn,
  GridSelection,
  HeaderClickedEventArgs,
  Item,
  Rectangle,
  Theme,
} from '@glideapps/glide-data-grid';
import { MenuProps } from 'antd';
import React, {
  Dispatch,
  SetStateAction,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useNavigate } from 'react-router';

import useUI from 'shared/contexts/stores/UI';
import usersStore from 'stores/users';
import { ExperimentItem, Project } from 'types';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

import { PAGE_SIZE } from '../F_ExperimentList';
import { MapOfIdsToColors } from '../useGlasbey';

import {
  ColumnDef,
  defaultColumnWidths,
  ExperimentColumn,
  getColumnDefs,
  getHeaderIcons,
} from './columns';
import { TableContextMenu, TableContextMenuProps } from './contextMenu';
import { customRenderers } from './custom-renderers';
import { LinkCell } from './custom-renderers/cells/linkCell';
import { placeholderMenuItems, TableActionMenu, TableActionMenuProps } from './menu';
import { useTableTooltip } from './tooltip';
import { getTheme } from './utils';

export interface GlideTableProps {
  clearSelectionTrigger?: number;
  colorMap: MapOfIdsToColors;
  data: Loadable<ExperimentItem>[];
  fetchExperiments: () => Promise<void>;
  handleScroll?: (r: Rectangle) => void;
  height: number;
  scrollPositionSetCount: WritableObservable<number>;
  sortableColumnIds: ExperimentColumn[];
  setSortableColumnIds: Dispatch<SetStateAction<ExperimentColumn[]>>;
  page: number;
  project?: Project;
  selectedExperimentIds: number[];
  setSelectedExperimentIds: Dispatch<SetStateAction<number[]>>;
  selectAll: boolean;
  setSelectAll: Dispatch<SetStateAction<boolean>>;
}

type ClickableCell = CustomCell<LinkCell> & {
  data: {
    link: {
      onClick: () => void;
    };
  };
};

/**
 * Number of renders with gridRef.current !== null
 * needed for the table to be properly initialized.
 * We set the scroll position to the persisted page
 * this many times, and then consider the scroll position to be
 * 'set' for purposes of the `handleScroll` in the parent component.
 * Otherwise handleScroll would erroneously set the page to 0
 * when the table is first initialized.
 */
export const SCROLL_SET_COUNT_NEEDED = 2;

const STATIC_COLUMNS: ExperimentColumn[] = ['selected', 'name'];

export const GlideTable: React.FC<GlideTableProps> = ({
  data,
  fetchExperiments,
  clearSelectionTrigger,
  setSelectedExperimentIds,
  sortableColumnIds,
  setSortableColumnIds,
  colorMap,
  height,
  selectAll,
  setSelectAll,
  handleScroll,
  scrollPositionSetCount,
  page,
  project,
}) => {
  const gridRef = useRef<DataEditorRef>(null);

  useEffect(() => {
    if (scrollPositionSetCount.get() >= SCROLL_SET_COUNT_NEEDED) return;
    if (gridRef.current !== null) {
      const rowOffset = Math.max(page * PAGE_SIZE, 0);
      gridRef.current.scrollTo(0, rowOffset);
      scrollPositionSetCount.update((x) => x + 1);
    }
  });

  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const handleMenuClose = useCallback(() => {
    setMenuIsOpen(false);
  }, []);
  const [menuProps, setMenuProps] = useState<Omit<TableActionMenuProps, 'open'>>({
    handleClose: handleMenuClose,
    x: 0,
    y: 0,
  });

  const [contextMenuOpen] = useState(observable(false));
  const contextMenuIsOpen = useObservable(contextMenuOpen);

  const [contextMenuProps, setContextMenuProps] = useState<null | Omit<
    TableContextMenuProps,
    'open' | 'fetchExperiments'
  >>(null);

  const {
    ui: { theme: appTheme, darkLight },
  } = useUI();
  const theme = getTheme(appTheme);

  const users = useObservable(usersStore.getUsers());

  const columnIds = useMemo<ExperimentColumn[]>(
    () => [...STATIC_COLUMNS, ...sortableColumnIds],
    [sortableColumnIds],
  );
  const navigate = useNavigate();

  const [selection, setSelection] = React.useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  });

  useEffect(() => {
    if (clearSelectionTrigger === 0) return;
    setSelection({ columns: CompactSelection.empty(), rows: CompactSelection.empty() });
  }, [clearSelectionTrigger]);

  useEffect(() => {
    const selectedRowIndices = selection.rows.toArray();
    setSelectedExperimentIds((prevIds) => {
      const selectedIds = selectedRowIndices
        .map((idx) => data?.[idx])
        .filter((row) => row !== undefined)
        .filter(Loadable.isLoaded)
        .map((record) => record.data.id);
      if (prevIds === selectedIds) return prevIds;
      return selectedIds;
    });
  }, [selection.rows, setSelectedExperimentIds, data]);

  const [columnWidths, setColumnWidths] =
    useState<Record<ExperimentColumn, number>>(defaultColumnWidths);

  const columnDefs = useMemo<Record<ExperimentColumn, ColumnDef>>(
    () =>
      getColumnDefs({
        appTheme,
        columnWidths,
        darkLight,
        navigate,
        rowSelection: selection.rows,
        selectAll,
        users,
      }),
    [navigate, selectAll, selection.rows, columnWidths, users, darkLight, appTheme],
  );

  const headerIcons = useMemo(() => getHeaderIcons(appTheme), [appTheme]);

  const { tooltip, onItemHovered, closeTooltip } = useTableTooltip({
    columnDefs,
    columnIds,
    data,
  });

  const getRowThemeOverride: DataEditorProps['getRowThemeOverride'] = React.useCallback(
    (row: number): Partial<Theme> | undefined => {
      const baseRowTheme = { borderColor: appTheme.stageStrong };
      // to put a border on the bottom row (actually the top of the row below it)
      if (row === data.length) return baseRowTheme;
      // avoid showing 'empty rows' below data
      if (!data[row]) return;
      const rowColorTheme = Loadable.match(data[row], {
        Loaded: (record) => (colorMap[record.id] ? { accentColor: colorMap[record.id] } : {}),
        NotLoaded: () => ({}),
      });
      return { ...baseRowTheme, ...rowColorTheme };
    },
    [colorMap, data, appTheme],
  );

  const onColumnResize: DataEditorProps['onColumnResize'] = useCallback(
    (column: GridColumn, width: number) => {
      const columnId = column.id;
      if (columnId === undefined || columnId === 'selected') return;
      setColumnWidths((prevWidths) => {
        const prevWidth = prevWidths[columnId as ExperimentColumn];
        if (width === prevWidth) return prevWidths;
        return { ...prevWidths, [columnId]: width };
      });
    },
    [],
  );

  const onColumnResizeEnd: DataEditorProps['onColumnResizeEnd'] = useCallback(() => {
    // presumably update the settings, but maybe have a different API
    // like Record<ColumnName, width>
  }, []);

  const onHeaderClicked: DataEditorProps['onHeaderClicked'] = React.useCallback(
    (col: number, args: HeaderClickedEventArgs) => {
      const columnId = columnIds[col];

      if (columnId === 'selected') {
        setSelectAll((prev) => !prev);
        return;
      }

      const { bounds } = args;
      const items: MenuProps['items'] = placeholderMenuItems;
      const x = bounds.x;
      const y = bounds.y + bounds.height;
      setMenuProps((prev) => ({ ...prev, items, title: `${columnId} menu`, x, y }));
      setMenuIsOpen(true);
    },
    [columnIds, setSelectAll],
  );

  const getCellContent: DataEditorProps['getCellContent'] = React.useCallback(
    (cell: Item): GridCell => {
      const [colIdx, rowIdx] = cell;
      const columnId = columnIds[colIdx];
      const row = data[rowIdx];
      if (row && Loadable.isLoaded(row)) {
        return columnDefs[columnId].renderer(row.data, rowIdx);
      }
      return {
        allowOverlay: true,
        copyData: '',
        data: {
          kind: 'spinner-cell',
        },
        kind: GridCellKind.Custom,
      };
    },
    [data, columnIds, columnDefs],
  );

  const onCellClicked: DataEditorProps['onCellClicked'] = useCallback(
    (cell: Item) => {
      const [col, row] = cell;
      if (row === undefined) return;

      const columnId = columnIds[col];
      const rowData = data[row];
      if (Loadable.isLoaded(rowData)) {
        const cell = columnDefs[columnId].renderer(rowData.data, row) as ClickableCell;
        if (String(cell?.data?.kind) === 'link-cell') {
          cell.data.link?.onClick?.();
          return;
        }
      }

      setSelection(({ rows }: GridSelection) => ({
        columns: CompactSelection.empty(),
        rows: rows.hasIndex(row) ? rows.remove(row) : rows.add(row),
      }));
    },
    [data, columnIds, columnDefs],
  );

  const onCellContextMenu: DataEditorProps['onCellContextMenu'] = useCallback(
    (cell: Item, event: CellClickedEventArgs) => {
      contextMenuOpen.set(false);
      const [, row] = cell;
      const experiment = Loadable.match(data?.[row], {
        Loaded: (record) => record,
        NotLoaded: () => null,
      });
      if (!experiment) return;

      event.preventDefault();
      setContextMenuProps({
        experiment: getProjectExperimentForExperimentItem(experiment, project),
        handleClose: (e?: Event) => {
          if (contextMenuOpen.get()) {
            e?.stopPropagation();
          }
          contextMenuOpen.set(false);
        },
        x: Math.max(0, event.bounds.x + event.localEventX - 4),
        y: Math.max(0, event.bounds.y + event.localEventY - 4),
      });
      setTimeout(() => contextMenuOpen.set(true), 25);
    },
    [data, project, setContextMenuProps, contextMenuOpen],
  );

  const onColumnMoved: DataEditorProps['onColumnMoved'] = useCallback(
    (columnIdsStartIdx: number, columnIdsEndIdx: number): void => {
      const sortableColumnIdsStartIdx = columnIdsStartIdx - STATIC_COLUMNS.length;
      const sortableColumnIdsEndIdx = Math.max(columnIdsEndIdx - STATIC_COLUMNS.length, 0);
      if (sortableColumnIdsStartIdx > -1) {
        setSortableColumnIds((prevCols) => {
          const newCols = [...prevCols];
          const [toMove] = newCols.splice(sortableColumnIdsStartIdx, 1);
          newCols.splice(sortableColumnIdsEndIdx, 0, toMove);
          return newCols;
        });
      }
    },
    [setSortableColumnIds],
  );

  const columns: DataEditorProps['columns'] = useMemo(
    () => columnIds.map((columnName) => columnDefs[columnName as ExperimentColumn]) as GridColumn[],
    [columnIds, columnDefs],
  );

  const verticalBorder: DataEditorProps['verticalBorder'] = useCallback(
    (col: number) => columnIds[col] === 'name',
    [columnIds],
  );

  return (
    <div
      onWheel={() => {
        contextMenuOpen.set(false);
        closeTooltip();
      }}>
      {tooltip}
      <DataEditor
        columns={columns}
        customRenderers={customRenderers}
        freezeColumns={2}
        getCellContent={getCellContent}
        getRowThemeOverride={getRowThemeOverride}
        gridSelection={selection}
        headerHeight={36}
        headerIcons={headerIcons}
        height={height}
        ref={gridRef}
        rowHeight={40}
        rows={data.length}
        smoothScrollX
        smoothScrollY
        theme={theme}
        verticalBorder={verticalBorder}
        width="100%"
        onCellClicked={onCellClicked}
        onCellContextMenu={onCellContextMenu}
        onColumnMoved={onColumnMoved}
        onColumnResize={onColumnResize}
        onColumnResizeEnd={onColumnResizeEnd}
        onHeaderClicked={onHeaderClicked}
        onItemHovered={onItemHovered}
        onVisibleRegionChanged={handleScroll}
      />
      <TableActionMenu {...menuProps} open={menuIsOpen} />
      {contextMenuProps && (
        <TableContextMenu
          {...contextMenuProps}
          fetchExperiments={fetchExperiments}
          open={contextMenuIsOpen}
        />
      )}
    </div>
  );
};

export default GlideTable;
