import DataEditor, {
  CompactSelection,
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
import userStore from 'stores/users';
import { ExperimentItem } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable, WritableObservable } from 'utils/observable';

import { PAGE_SIZE } from '../F_ExperimentList';

import { ColumnDef, defaultColumnWidths, ExperimentColumn, getColumnDefs } from './columns';
import UserProfileCell from './custom-cells/avatar';
import LinksCell from './custom-cells/links';
import RangeCell from './custom-cells/progress';
import SparklineCell from './custom-cells/sparkline';
import SpinnerCell from './custom-cells/spinner';
import TagsCell from './custom-cells/tags';
import { placeholderMenuItems, TableActionMenu, TableActionMenuProps } from './menu';
import { MapOfIdsToColors } from './useGlasbey';
import { getTheme, headerIcons } from './utils';

const GRID_HEIGHT = 700;
const cells: DataEditorProps['customRenderers'] = [
  SparklineCell,
  TagsCell,
  UserProfileCell,
  SpinnerCell,
  RangeCell,
  LinksCell,
];

interface Props {
  colorMap: MapOfIdsToColors;
  data: Loadable<ExperimentItem>[];
  handleScroll?: (r: Rectangle) => void;
  initialScrollPositionSet: WritableObservable<boolean>;
  sortableColumnIds: ExperimentColumn[];
  setSortableColumnIds: Dispatch<SetStateAction<ExperimentColumn[]>>;
  page: number;
  selectedExperimentIds: string[];
  setSelectedExperimentIds: Dispatch<SetStateAction<string[]>>;
  selectAll: boolean;
  setSelectAll: Dispatch<SetStateAction<boolean>>;
}

const STATIC_COLUMNS: ExperimentColumn[] = ['selected', 'name'];

export const GlideTable: React.FC<Props> = ({
  data,
  setSelectedExperimentIds,
  sortableColumnIds,
  setSortableColumnIds,
  colorMap,
  selectAll,
  setSelectAll,
  handleScroll,
  initialScrollPositionSet,
  page,
}) => {
  const gridRef = useRef<DataEditorRef>(null);

  useEffect(() => {
    if (initialScrollPositionSet.get()) return;
    setTimeout(() => {
      if (gridRef.current !== null) {
        const rowOffset = Math.max(page * PAGE_SIZE, 0);
        gridRef.current.scrollTo(0, rowOffset);
        setTimeout(() => initialScrollPositionSet.set(true), 200);
      }
    }, 200);
  }, [initialScrollPositionSet, page]);

  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const handleMenuClose = useCallback(() => {
    setMenuIsOpen(false);
  }, []);
  const [menuProps, setMenuProps] = useState<Omit<TableActionMenuProps, 'open'>>({
    handleClose: handleMenuClose,
    x: 0,
    y: 0,
  });

  const {
    ui: { darkLight },
  } = useUI();

  const users = useObservable(userStore.getUsers());

  const columnIds = useMemo<ExperimentColumn[]>(
    () => [...STATIC_COLUMNS, ...sortableColumnIds],
    [sortableColumnIds],
  );
  const navigate = useNavigate();
  const bodyStyles = getComputedStyle(document.body);

  const [selection, setSelection] = React.useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  });

  const getRowThemeOverride = React.useCallback(
    (row: number): Partial<Theme> | undefined => {
      if (!data[row]) return;
      const accentColor = Loadable.match(data[row], {
        Loaded: (record) => (colorMap[record.id] ? { accentColor: colorMap[record.id] } : {}),
        NotLoaded: () => ({}),
      });
      return { borderColor: '#F0F0F0', ...accentColor };
    },
    [colorMap, data],
  );

  useEffect(() => {
    const selectedRowIndices = selection.rows.toArray();
    setSelectedExperimentIds((prevIds) => {
      const selectedIds = selectedRowIndices
        .map((idx) => data?.[idx])
        .filter(Loadable.isLoaded)
        .map((record) => String(record.data.id));
      if (prevIds === selectedIds) return prevIds;
      return selectedIds;
    });
  }, [selection.rows, setSelectedExperimentIds, data]);

  const theme = getTheme(bodyStyles);

  const [columnWidths, setColumnWidths] =
    useState<Record<ExperimentColumn, number>>(defaultColumnWidths);

  const columnDefs = useMemo<Record<ExperimentColumn, ColumnDef>>(
    () =>
      getColumnDefs({
        bodyStyles,
        columnWidths,
        darkLight,
        navigate,
        rowSelection: selection.rows,
        selectAll,
        users,
      }),
    /**
     * dont have a stable reference to bodyStyles
     * presumably we capture whatever changes we need when darkLight
     * changes though (since that changes the theme vars)
     */
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [navigate, selectAll, selection.rows, columnWidths, users, darkLight],
  );

  const onColumnResize = useCallback((column: GridColumn, width: number) => {
    const columnId = column.id;
    if (columnId === undefined || columnId === 'selected') return;
    setColumnWidths((prevWidths) => {
      const prevWidth = prevWidths[columnId as ExperimentColumn];
      if (width === prevWidth) return prevWidths;
      return { ...prevWidths, [columnId]: width };
    });
  }, []);

  const onColumnResizeEnd = useCallback(() => {
    // presumably update the settings, but maybe have a different API
    // like Record<ColumnName, width>
  }, []);

  const onHeaderClicked = React.useCallback(
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

  const getCellContent = React.useCallback(
    (cell: Item): GridCell => {
      const [colIdx, rowIdx] = cell;
      const columnId = columnIds[colIdx];
      const row = data[rowIdx];
      if (Loadable.isLoaded(row)) {
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

  const handleGridSelectionChange = useCallback((newSelection: GridSelection) => {
    const [, row] = newSelection.current?.cell ?? [undefined, undefined];
    if (row === undefined) return;
    setSelection(({ rows }: GridSelection) => ({
      columns: CompactSelection.empty(),
      rows: rows.hasIndex(row) ? rows.remove(row) : rows.add(row),
    }));
  }, []);

  const onColumnMoved = useCallback(
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

  const dataGridColumns = useMemo(
    () => columnIds.map((columnName) => columnDefs[columnName as ExperimentColumn]) as GridColumn[],
    [columnIds, columnDefs],
  );

  const verticalBorder = useCallback((col: number) => columnIds[col] === 'name', [columnIds]);

  return (
    <div>
      <DataEditor
        columns={dataGridColumns}
        customRenderers={cells}
        freezeColumns={2}
        getCellContent={getCellContent}
        getRowThemeOverride={getRowThemeOverride}
        gridSelection={selection}
        headerIcons={headerIcons}
        height={GRID_HEIGHT}
        ref={gridRef}
        rows={data.length}
        smoothScrollX
        smoothScrollY
        theme={theme}
        verticalBorder={verticalBorder}
        width="100%"
        onColumnMoved={onColumnMoved}
        onColumnResize={onColumnResize}
        onColumnResizeEnd={onColumnResizeEnd}
        onGridSelectionChange={handleGridSelectionChange}
        onHeaderClicked={onHeaderClicked}
        onVisibleRegionChanged={handleScroll}
        //
        // these might come in handy
        // onCellClicked={onCellClicked}
        // onItemHovered={onItemHovered}
        // onHeaderContextMenu={onHeaderContextMenu}
        // onCellContextMenu={onCellContextMenu}
      />
      <TableActionMenu {...menuProps} open={menuIsOpen} />
    </div>
  );
};

export default GlideTable;
