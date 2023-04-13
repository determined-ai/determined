import DataEditor, {
  CellClickedEventArgs,
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
import css from './index.module.scss';
import {
  contextMenuItems,
  pinnedContextMenuItems,
  placeholderMenuItems,
  TableActionMenu,
  TableActionMenuProps,
} from './menu';
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
  const pinnedGridRef = useRef<DataEditorRef>(null);

  const [pinnedRows, setPinnedRows] = useState<Loadable<ExperimentItem>[]>([]);
  const [mainTableData, setMainTableData] = useState<Loadable<ExperimentItem>[]>(data);
  const [originalIndex, setOriginalIndex] = useState<number[]>([]);
  const [mainGridScroll, setMainGridScroll] = useState(0);
  const [pinnedGridScroll, setPinnedGridScroll] = useState(0);

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

  const scrollGrid = (ref: React.RefObject<DataEditorRef>, amount: number) => {
    if (!ref.current) return;
    ref.current.scrollTo({ amount, unit: 'cell' }, 0, 'horizontal', 0, 0, {
      hAlign: 'start',
      vAlign: 'start',
    });
  };

  useEffect(() => {
    scrollGrid(pinnedGridRef, mainGridScroll);
  }, [mainGridScroll]);

  useEffect(() => {
    scrollGrid(gridRef, pinnedGridScroll);
  }, [pinnedGridScroll]);

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
      if (!mainTableData[row]) return;
      const accentColor = Loadable.match(mainTableData[row], {
        Loaded: (record) => (colorMap[record.id] ? { accentColor: colorMap[record.id] } : {}),
        NotLoaded: () => ({}),
      });
      return { borderColor: '#F0F0F0', ...accentColor };
    },
    [colorMap, mainTableData],
  );

  useEffect(() => {
    const selectedRowIndices = selection.rows.toArray();
    setSelectedExperimentIds((prevIds) => {
      const selectedIds = selectedRowIndices
        .map((idx) => mainTableData?.[idx])
        .filter(Loadable.isLoaded)
        .map((record) => String(record.data.id));
      if (prevIds === selectedIds) return prevIds;
      return selectedIds;
    });
  }, [selection.rows, setSelectedExperimentIds, mainTableData]);

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
      if (pinnedRows.length) return;

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
    [columnIds, setSelectAll, pinnedRows.length],
  );

  const getCellContent = React.useCallback(
    (cell: Item): GridCell => {
      const [colIdx, rowIdx] = cell;
      const columnId = columnIds[colIdx];
      const row = mainTableData[rowIdx];
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
    [mainTableData, columnIds, columnDefs],
  );

  const getPinnedCellContent = React.useCallback(
    (cell: Item): GridCell => {
      const [colIdx, rowIdx] = cell;
      const columnId = columnIds[colIdx];
      const row = pinnedRows[rowIdx];
      if (Loadable.isLoaded(row) && row !== undefined) {
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
    [pinnedRows, columnIds, columnDefs],
  );

  const onContextMenu = React.useCallback(
    (args: CellClickedEventArgs) => {
      args.preventDefault();

      const { bounds } = args;
      const items: MenuProps['items'] = contextMenuItems;
      const x = bounds.x;
      const y = bounds.y + bounds.height;
      setMenuProps((prev) => ({
        ...prev,
        handleClick: () => {
          const rowIndex = args.location[1];
          const originalData = data.map((row) =>
            Loadable.isLoaded(row) ? row.data : ({} as ExperimentItem),
          );
          const mainData = mainTableData.map((row) =>
            Loadable.isLoaded(row) ? row.data : ({} as ExperimentItem),
          );

          setOriginalIndex((prev) => {
            prev.push(originalData.findIndex((row) => row.id === mainData[rowIndex].id));
            return [...prev];
          });

          setPinnedRows((prev) => {
            prev.push(mainTableData[rowIndex]);
            return [...prev];
          });

          setMainTableData((prev) => {
            prev.splice(rowIndex, 1);

            return [...prev];
          });

          setMenuIsOpen(false);
        },
        isContextMenu: true,
        items,
        title: '',
        x,
        y,
      }));
      setMenuIsOpen(true);
    },
    [mainTableData, data],
  );

  const onPinnedGridContextMenu = React.useCallback(
    (args: CellClickedEventArgs) => {
      args.preventDefault();

      const { bounds } = args;
      const items: MenuProps['items'] = pinnedContextMenuItems;
      const x = bounds.x;
      const y = bounds.y + bounds.height;
      setMenuProps((prev) => ({
        ...prev,
        handleClick: () => {
          const rowIndex = args.location[1];
          const prevIndex = originalIndex[rowIndex];
          setOriginalIndex((prev) => {
            prev.splice(rowIndex, 1);
            return [...prev];
          });

          setMainTableData((prev) => {
            prev.splice(prevIndex, 0, data[prevIndex]);
            return [...prev];
          });

          setPinnedRows((prev) => {
            prev.splice(rowIndex, 1);

            return [...prev];
          });

          setMenuIsOpen(false);
        },
        isContextMenu: true,
        items,
        title: '',
        x,
        y,
      }));
      setMenuIsOpen(true);
    },
    [originalIndex, data],
  );

  const onCellClicked = React.useCallback(() => {
    if (menuIsOpen) setMenuIsOpen(false);
  }, [menuIsOpen]);

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

  const onScroll = React.useCallback(
    (range: Rectangle) => {
      if (pinnedRows.length !== 0 && range.x !== mainGridScroll) {
        setMainGridScroll(range.x);
      }
      handleScroll?.(range);
    },
    [pinnedRows.length, mainGridScroll, handleScroll],
  );

  const onScrollPinnedGrid = React.useCallback(
    (range: Rectangle) => {
      if (pinnedRows.length !== 0 && range.x !== mainGridScroll) {
        setPinnedGridScroll(range.x);
      }
      handleScroll?.(range);
    },
    [pinnedRows.length, mainGridScroll, handleScroll],
  );

  return (
    <div>
      {!!pinnedRows.length && (
        <DataEditor
          className={css.pinnedGrid}
          columns={dataGridColumns}
          customRenderers={cells}
          freezeColumns={2}
          getCellContent={getPinnedCellContent}
          headerIcons={headerIcons}
          ref={pinnedGridRef}
          rows={pinnedRows.length}
          theme={theme}
          verticalBorder={verticalBorder}
          width="100%"
          onCellClicked={onCellClicked}
          onCellContextMenu={(_, event) => onPinnedGridContextMenu(event)}
          onVisibleRegionChanged={onScrollPinnedGrid}
        />
      )}
      <DataEditor
        columns={dataGridColumns}
        customRenderers={cells}
        freezeColumns={2}
        getCellContent={getCellContent}
        getRowThemeOverride={getRowThemeOverride}
        gridSelection={selection}
        headerHeight={pinnedRows.length !== 0 ? 0 : undefined}
        headerIcons={headerIcons}
        height={GRID_HEIGHT}
        ref={gridRef}
        rows={mainTableData.length}
        smoothScrollX={!pinnedRows.length}
        smoothScrollY={!pinnedRows.length}
        theme={theme}
        verticalBorder={verticalBorder}
        width="100%"
        onCellClicked={onCellClicked}
        onCellContextMenu={(_, event) => onContextMenu(event)}
        onColumnMoved={onColumnMoved}
        onColumnResize={onColumnResize}
        onColumnResizeEnd={onColumnResizeEnd}
        onGridSelectionChange={handleGridSelectionChange}
        onHeaderClicked={onHeaderClicked}
        onVisibleRegionChanged={onScroll}
        //
        // these might come in handy
        // onItemHovered={onItemHovered}
        // onHeaderContextMenu={onHeaderContextMenu}
      />
      <TableActionMenu {...menuProps} open={menuIsOpen} />
    </div>
  );
};

export default GlideTable;
