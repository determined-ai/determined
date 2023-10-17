import DataEditor, {
  CellClickedEventArgs,
  DataEditorProps,
  DataEditorRef,
  getMiddleCenterBias,
  GridCell,
  GridCellKind,
  GridColumn,
  GridMouseEventArgs,
  GridSelection,
  HeaderClickedEventArgs,
  Item,
  Rectangle,
  Theme,
} from '@hpe.com/glide-data-grid';
import { DrawHeaderCallback } from '@hpe.com/glide-data-grid/dist/ts/data-grid/data-grid-types';
import { literal, union } from 'io-ts';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';

import { FilterFormStore, ROOT_ID } from 'components/FilterForm/components/FilterFormStore';
import {
  AvailableOperators,
  FormKind,
  Operator,
  SpecialColumnNames,
} from 'components/FilterForm/components/type';
import { DropdownEvent, MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import useUI, { getCssVar } from 'components/kit/Theme';
import { Loadable } from 'components/kit/utils/loadable';
import { MapOfIdsToColors } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import { type HandleSelectionChangeType, PAGE_SIZE } from 'pages/F_ExpList/F_ExperimentList';
import { handlePath } from 'routes/utils';
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import usersStore from 'stores/users';
import {
  ExperimentAction,
  ExperimentItem,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  ProjectMetricsRange,
} from 'types';
import { Float, Surface } from 'utils/colors';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';
import { observable, useObservable, WritableObservable } from 'utils/observable';
import { AnyMouseEvent } from 'utils/routes';

import {
  ColumnDef,
  defaultDateColumn,
  defaultNumberColumn,
  defaultTextColumn,
  getColumnDefs,
  getHeaderIcons,
  MIN_COLUMN_WIDTH,
  MULTISELECT,
  searcherMetricsValColumn,
} from './columns';
import { TableContextMenu, TableContextMenuProps } from './contextMenu';
import { customRenderers } from './custom-renderers';
import { LinkCell } from './custom-renderers/cells/linkCell';
import { drawArrow, drawTextWithEllipsis } from './custom-renderers/utils';
import css from './GlideTable.module.scss';
import { TableActionMenu, TableActionMenuProps } from './menu';
import { Sort, sortMenuItemsForColumn } from './MultiSortMenu';
import { RowHeight } from './OptionsMenu';
import { useTableTooltip } from './tooltip';
import { getTheme } from './utils';

export interface GlideTableProps {
  colorMap: MapOfIdsToColors;
  columnWidths: Record<string, number>;
  comparisonViewOpen?: boolean;
  data: Loadable<ExperimentWithTrial>[];
  dataTotal: number;
  formStore: FilterFormStore;
  heatmapOn: boolean;
  heatmapSkipped: string[];
  height: number;
  onColumnResize?: (newColumnWidths: Record<string, number>) => void;
  onContextMenuComplete?: (
    action: ExperimentAction,
    id: number,
    data?: Partial<ExperimentItem>,
  ) => void;
  onHeatmapSelection?: (selection: string[]) => void;
  onIsOpenFilterChange?: (value: boolean) => void;
  onPinnedColumnsCountChange?: (count: number) => void;
  onScroll?: (r: Rectangle) => void;
  onSelectionChange?: HandleSelectionChangeType;
  onSortableColumnChange?: (newColumns: string[]) => void;
  onSortChange?: (sorts: Sort[]) => void;
  page: number;
  pinnedColumnsCount: number;
  project?: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  projectHeatmap: ProjectMetricsRange[];
  rowHeight: RowHeight;
  scrollPositionSetCount: WritableObservable<number>;
  selectAll: boolean;
  selection: GridSelection;
  sortableColumnIds: string[];
  sorts: Sort[];
  staticColumns: string[];
}

export type TableViewMode = 'scroll' | 'paged';

export const ioTableViewMode = union([literal('scroll'), literal('paged')]);

/**
 * Number of renders with gridRef.current !== null
 * needed for the table to be properly initialized.
 * We set the scroll position to the persisted page
 * this many times, and then consider the scroll position to be
 * 'set' for purposes of the `onScroll` in the parent component.
 * Otherwise `onScroll` would erroneously set the page to 0
 * when the table is first initialized.
 */
export const SCROLL_SET_COUNT_NEEDED = 3;

const isLinkCell = (cell: GridCell): cell is LinkCell => {
  return !!(cell as LinkCell).data?.link?.href;
};

const rowHeightMap: Record<RowHeight, number> = {
  [RowHeight.EXTRA_TALL]: 44,
  [RowHeight.TALL]: 40,
  [RowHeight.MEDIUM]: 36,
  [RowHeight.SHORT]: 32,
};

export const GlideTable: React.FC<GlideTableProps> = ({
  colorMap,
  columnWidths,
  comparisonViewOpen = false,
  data,
  dataTotal,
  formStore,
  heatmapOn,
  heatmapSkipped,
  height,
  onColumnResize,
  onContextMenuComplete,
  onHeatmapSelection,
  onIsOpenFilterChange,
  onPinnedColumnsCountChange,
  onScroll,
  onSelectionChange,
  onSortableColumnChange,
  onSortChange,
  page,
  pinnedColumnsCount,
  project,
  projectColumns,
  projectHeatmap,
  rowHeight,
  scrollPositionSetCount,
  selectAll,
  selection,
  sortableColumnIds,
  sorts,
  staticColumns,
}) => {
  const gridRef = useRef<DataEditorRef>(null);
  const clickedCellRef = useRef<{ col: number; row: number } | null>(null);
  const [hoveredRow, setHoveredRow] = useState<number>();
  const elementRef = useRef(null)
  useEffect(() => {
    if (scrollPositionSetCount.get() >= SCROLL_SET_COUNT_NEEDED) return;
    if (gridRef.current !== null) {
      const rowOffset = Math.max(page * PAGE_SIZE, 0);
      gridRef.current.scrollTo(0, rowOffset);
      scrollPositionSetCount.update((x) => x + 1);
    }
  });

  const [menuIsOpen, setMenuIsOpen] = useState(false);
  const [menuProps, setMenuProps] = useState<Omit<TableActionMenuProps, 'open'>>({
    bounds: {
      height: 0,
      width: 0,
      x: 0,
      y: 0,
    },
    handleClose: () => setMenuIsOpen(false),
  });

  const handleContextMenuComplete = useCallback(
    (action: ExperimentAction, id: number, data?: Partial<ExperimentItem>) => {
      onContextMenuComplete?.(action, id, data);
    },
    [onContextMenuComplete],
  );

  const [contextMenuOpen] = useState(observable(false));
  const contextMenuIsOpen = useObservable(contextMenuOpen);

  const [contextMenuProps, setContextMenuProps] = useState<null | Omit<
    TableContextMenuProps,
    'open'
  >>(null);

  const {
    ui: { theme: appTheme, darkLight },
  } = useUI();
  const theme = getTheme(appTheme);

  const users = useObservable(usersStore.getUsers());

  const isMobile = useMobile();

  const columnIds = useMemo(
    () =>
      comparisonViewOpen
        ? [...staticColumns, ...sortableColumnIds.slice(0, pinnedColumnsCount)]
        : [...staticColumns, ...sortableColumnIds],
    [comparisonViewOpen, pinnedColumnsCount, sortableColumnIds, staticColumns],
  );

  const columnDefs = useMemo<Record<string, ColumnDef>>(
    () =>
      getColumnDefs({
        appTheme,
        columnWidths,
        darkLight,
        rowSelection: selection.rows,
        selectAll,
        users,
      }),
    [selectAll, selection.rows, columnWidths, users, darkLight, appTheme],
  );

  const headerIcons = useMemo(() => getHeaderIcons(appTheme), [appTheme]);

  const projectColumnsMap: Loadable<Record<string, ProjectColumn>> = useMemo(() => {
    return Loadable.map(projectColumns, (columns) => {
      return columns.reduce((acc, col) => ({ ...acc, [col.column]: col }), {});
    });
  }, [projectColumns]);

  const { tooltip, onItemHovered, closeTooltip } = useTableTooltip({
    columnDefs,
    columnIds,
    data,
  });

  const getRowThemeOverride: DataEditorProps['getRowThemeOverride'] = React.useCallback(
    (row: number): Partial<Theme> | undefined => {
      // to put a border on the bottom row (actually the top of the row below it)
      if (row === data.length) return;

      // avoid showing 'empty rows' below data
      if (!data[row]) {
        return { borderColor: getCssVar(elementRef, Surface.Surface) };
      }

      const hoverStyle: { accentLight?: string; bgCell?: string } = {};
      if (row === hoveredRow) {
        hoverStyle.bgCell = getCssVar(elementRef, Surface.SurfaceStrong);
        if (selection.rows.toArray().includes(hoveredRow)) {
          hoverStyle.accentLight = getCssVar(elementRef, Float.FloatStrong);
        }
      }

      const rowColorTheme = Loadable.match(data[row], {
        _: () => ({}),
        Loaded: (record) =>
          colorMap[record.experiment.id] ? { accentColor: colorMap[record.experiment.id] } : {},
      });

      return { ...rowColorTheme, ...hoverStyle };
    },
    [colorMap, data, hoveredRow, selection.rows],
  );

  const handleColumnResize: DataEditorProps['onColumnResize'] = useCallback(
    (column: GridColumn, width: number) => {
      const columnId = column.id;
      if (columnId === undefined || columnId === MULTISELECT) return;
      onColumnResize?.({ ...columnWidths, [columnId]: width });
    },
    [columnWidths, onColumnResize],
  );

  const toggleHeatmap = useCallback(
    (col: string) =>
      onHeatmapSelection?.(
        heatmapSkipped.includes(col)
          ? heatmapSkipped.filter((p) => p !== col)
          : [...heatmapSkipped, col],
      ),
    [heatmapSkipped, onHeatmapSelection],
  );

  const onHeaderClicked: DataEditorProps['onHeaderClicked'] = React.useCallback(
    (col: number, { bounds, preventDefault }: HeaderClickedEventArgs) => {
      preventDefault();
      const columnId = columnIds[col];

      if (columnId === MULTISELECT) {
        const items: MenuItem[] = [
          selection.rows.length > 0
            ? {
              key: 'select-none',
              label: 'Clear selected',
              onClick: () => {
                onSelectionChange?.('remove-all', [0, data.length]);
                setMenuIsOpen(false);
              },
            }
            : null,
          ...[5, 10, 25].map((n) => ({
            key: `select-${n}`,
            label: `Select first ${n}`,
            onClick: () => {
              onSelectionChange?.('set', [0, n]);
              if (gridRef.current) {
                // scroll first row into view for feedback
                gridRef.current.scrollTo(0, 0);
              }
              setMenuIsOpen(false);
            },
          })),
          {
            key: 'select-all',
            label: 'Select all',
            onClick: () => {
              onSelectionChange?.('add-all', [0, data.length]);
              setMenuIsOpen(false);
            },
          },
        ];
        setMenuProps((prev) => ({ ...prev, bounds, items, title: 'Selection menu' }));
        setMenuIsOpen(true);
        return;
      }
      const column = Loadable.getOrElse([], projectColumns).find((c) => c.column === columnId);
      if (!column) {
        return;
      }

      const BANNED_FILTER_COLUMNS = ['searcherMetricsVal'];
      const loadableFormset = formStore.formset.get();
      const filterMenuItemsForColumn = () => {
        const isSpecialColumn = (SpecialColumnNames as ReadonlyArray<string>).includes(
          column.column,
        );
        formStore.addChild(ROOT_ID, FormKind.Field, {
          index: Loadable.match(loadableFormset, {
            _: () => 0,
            Loaded: (formset) => formset.filterGroup.children.length,
          }),
          item: {
            columnName: column.column,
            id: uuidv4(),
            kind: FormKind.Field,
            location: column.location,
            operator: isSpecialColumn ? Operator.Eq : AvailableOperators[column.type][0],
            type: column.type,
            value: null,
          },
        });
        onIsOpenFilterChange?.(true);
        setMenuIsOpen(false);
      };

      const items: MenuItem[] = [
        ...(BANNED_FILTER_COLUMNS.includes(column.column)
          ? []
          : [
            ...sortMenuItemsForColumn(column, sorts, onSortChange),
            { type: 'divider' as const },
            {
              icon: <Icon decorative name="filter" />,
              key: 'filter',
              label: 'Filter by this column',
              onClick: () => {
                setTimeout(() => {
                  filterMenuItemsForColumn();
                }, 5);
              },
            },
          ]),
        heatmapOn &&
          (column.column === 'searcherMetricsVal' ||
            (column.type === V1ColumnType.NUMBER &&
              (column.location === V1LocationType.VALIDATIONS ||
                column.location === V1LocationType.TRAINING)))
          ? {
            icon: <Icon decorative name="heatmap" />,
            key: 'heatmap',
            label: !heatmapSkipped.includes(column.column) ? 'Cancel heatmap' : 'Apply heatmap',
            onClick: () => {
              toggleHeatmap(column.column);
            },
          }
          : null,
        // Column is pinned if the index is inside of the frozen columns
        col < staticColumns.length || isMobile
          ? null
          : col > pinnedColumnsCount + staticColumns.length - 1
            ? {
              key: 'pin',
              label: 'Pin column',
              onClick: () => {
                const newSortableColumns = sortableColumnIds.filter((c) => c !== column.column);
                newSortableColumns.splice(pinnedColumnsCount, 0, column.column);
                onSortableColumnChange?.(newSortableColumns);
                onPinnedColumnsCountChange?.(
                  Math.min(pinnedColumnsCount + 1, sortableColumnIds.length),
                );
                setMenuIsOpen(false);
              },
            }
            : {
              disabled: pinnedColumnsCount <= 1,
              key: 'unpin',
              label: 'Unpin column',
              onClick: () => {
                const newSortableColumns = sortableColumnIds.filter((c) => c !== column.column);
                newSortableColumns.splice(pinnedColumnsCount - 1, 0, column.column);
                onSortableColumnChange?.(newSortableColumns);
                onPinnedColumnsCountChange?.(Math.max(pinnedColumnsCount - 1, 0));
                setMenuIsOpen(false);
              },
            },
      ];
      setMenuProps((prev) => ({ ...prev, bounds, items, title: `${columnId} menu` }));
      setMenuIsOpen(true);
    },
    [
      columnIds,
      data.length,
      projectColumns,
      formStore,
      sorts,
      onSortChange,
      staticColumns.length,
      isMobile,
      pinnedColumnsCount,
      selection.rows.length,
      onIsOpenFilterChange,
      onPinnedColumnsCountChange,
      onSelectionChange,
      onSortableColumnChange,
      sortableColumnIds,
      heatmapSkipped,
      toggleHeatmap,
      heatmapOn,
    ],
  );

  const getCellContent: DataEditorProps['getCellContent'] = React.useCallback(
    (cell: Item): GridCell => {
      const [col, row] = cell;

      const loadingCell: GridCell = {
        allowOverlay: true,
        copyData: '',
        data: { appTheme, kind: 'loading-cell' },
        kind: GridCellKind.Custom,
        readonly: true,
      };

      if (!data[row]) {
        // When data length is changed, data[row] can be undefined
        return loadingCell;
      }

      return Loadable.match(data[row], {
        _: () => loadingCell,
        Loaded: (rowData) => {
          const columnId = columnIds[col];
          return columnDefs[columnId]?.renderer?.(rowData, row) || loadingCell;
        }, // TODO correctly handle error state
      });
    },
    [appTheme, data, columnIds, columnDefs],
  );

  const onCellClicked: DataEditorProps['onCellClicked'] = useCallback(
    (cell: Item, event: CellClickedEventArgs) => {
      const [col, row] = cell;

      Loadable.forEach(data[row], (rowData) => {
        const columnId = columnIds[col];
        const cell = columnDefs[columnId].renderer(rowData, row);

        if (isLinkCell(cell)) {
          handlePath(event as unknown as AnyMouseEvent, { path: cell.data.link.href });
        } else {
          if (event.shiftKey) {
            if (clickedCellRef.current !== null) {
              const previousRow = clickedCellRef.current.row;
              const selectionType = selection.rows.toArray().includes(row) ? 'remove' : 'add';
              const range: [number, number] =
                previousRow < row ? [previousRow, row + 1] : [row, previousRow + 1];
              onSelectionChange?.(selectionType, range);
            }
          } else {
            if (selection.rows.hasIndex(row)) {
              onSelectionChange?.('remove', [row, row + 1]);
            } else {
              onSelectionChange?.('add', [row, row + 1]);
            }
          }
          clickedCellRef.current = { col, row };
        }
      });
    },
    [data, columnIds, columnDefs, onSelectionChange, selection],
  );

  const onCellContextMenu: DataEditorProps['onCellContextMenu'] = useCallback(
    (cell: Item, event: CellClickedEventArgs) => {
      const [col, row] = cell;

      // Close existing context menu.
      contextMenuOpen.set(false);

      Loadable.forEach(data[row], (rowData) => {
        // Prevent the browser native context menu from showing up.
        event.preventDefault();

        // Delay needed due to the call to close previously existing context menu.
        setTimeout(() => {
          const columnId = columnIds[col];
          const cell = columnDefs[columnId].renderer(rowData, row);

          // Update the context menu based on the cell context.
          setContextMenuProps({
            cell,
            experiment: getProjectExperimentForExperimentItem(rowData.experiment, project),
            link: isLinkCell(cell) ? cell.data.link.href : undefined,
            onClose: (e?: DropdownEvent | Event) => {
              // Prevent the context menu closing click from triggering something else.
              if (contextMenuOpen.get()) e?.stopPropagation();
              contextMenuOpen.set(false);
            },
            x: Math.max(0, event.bounds.x + event.localEventX - 4),
            y: Math.max(0, event.bounds.y + event.localEventY - 4),
          });

          contextMenuOpen.set(true);
        }, 50);
      });
    },
    [columnDefs, columnIds, data, project, setContextMenuProps, contextMenuOpen],
  );

  const onColumnMoved: DataEditorProps['onColumnMoved'] = useCallback(
    (columnIdsStartIdx: number, columnIdsEndIdx: number): void => {
      // Prevent the static columns from moving.
      if (columnIdsStartIdx < staticColumns.length) return;

      // Update the pinned column count based on where the column is sourced from and where it lands.
      const pinnedColumnEnd = staticColumns.length + pinnedColumnsCount;
      const isIntoPinned =
        columnIdsStartIdx >= pinnedColumnEnd && columnIdsEndIdx < pinnedColumnEnd;
      const isOutOfPinned =
        columnIdsStartIdx < pinnedColumnEnd && columnIdsEndIdx >= pinnedColumnEnd;
      if (isIntoPinned) onPinnedColumnsCountChange?.(pinnedColumnsCount + 1);
      if (isOutOfPinned) onPinnedColumnsCountChange?.(pinnedColumnsCount - 1);

      // Update the column list with the updated column.
      const sortableColumnIdsStartIdx = columnIdsStartIdx - staticColumns.length;
      const sortableColumnIdsEndIdx = Math.max(columnIdsEndIdx - staticColumns.length, 0);
      const newCols = [...sortableColumnIds];
      const [toMove] = newCols.splice(sortableColumnIdsStartIdx, 1);
      newCols.splice(sortableColumnIdsEndIdx, 0, toMove);
      onSortableColumnChange?.(newCols);
    },
    [
      onPinnedColumnsCountChange,
      onSortableColumnChange,
      pinnedColumnsCount,
      sortableColumnIds,
      staticColumns.length,
    ],
  );

  const onColumnHovered = useCallback(
    (args: GridMouseEventArgs) => {
      const [, row] = args.location;
      setHoveredRow(args.kind !== 'cell' ? undefined : row);
      onItemHovered?.(args);
    },
    [onItemHovered],
  );

  const columns: DataEditorProps['columns'] = useMemo(() => {
    const gridColumns = columnIds
      .map((columnName) => {
        if (columnName in columnDefs) return columnDefs[columnName];
        if (!Loadable.isLoaded(projectColumnsMap)) return;
        const currentColumn = projectColumnsMap.data[columnName];
        if (!currentColumn) return;
        let dataPath: string | undefined = undefined;
        switch (currentColumn.location) {
          case V1LocationType.EXPERIMENT:
            dataPath = `experiment.${currentColumn.column}`;
            break;
          case V1LocationType.HYPERPARAMETERS:
            dataPath = `experiment.config.hyperparameters.${currentColumn.column.replace(
              'hp.',
              '',
            )}.val`;
            break;
          case V1LocationType.VALIDATIONS:
            dataPath = `bestTrial.summaryMetrics.validationMetrics.${currentColumn.column.replace(
              'validation.',
              '',
            )}`;
            break;
          case V1LocationType.TRAINING:
            dataPath = `bestTrial.summaryMetrics.avgMetrics.${currentColumn.column.replace(
              'training.',
              '',
            )}`;
            break;
          case V1LocationType.CUSTOMMETRIC:
            dataPath = `bestTrial.summaryMetrics.${currentColumn.column}`;
            break;
          case V1LocationType.UNSPECIFIED:
            break;
        }
        switch (currentColumn.type) {
          case V1ColumnType.NUMBER: {
            const heatmap = projectHeatmap.find((h) => h.metricsName === currentColumn.column);
            if (heatmap && heatmapOn && !heatmapSkipped.includes(currentColumn.column)) {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn,
                columnWidths[currentColumn.column],
                dataPath,
                {
                  max: heatmap.max,
                  min: heatmap.min,
                },
              );
            } else {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn,
                columnWidths[currentColumn.column],
                dataPath,
              );
            }
            break;
          }
          case V1ColumnType.DATE:
            columnDefs[currentColumn.column] = defaultDateColumn(
              currentColumn,
              columnWidths[currentColumn.column],
              dataPath,
            );
            break;
          case V1ColumnType.TEXT:
          case V1ColumnType.UNSPECIFIED:
          default:
            columnDefs[currentColumn.column] = defaultTextColumn(
              currentColumn,
              columnWidths[currentColumn.column],
              dataPath,
            );
        }
        if (currentColumn.column === 'searcherMetricsVal') {
          const heatmap = projectHeatmap.find((h) => h.metricsName === currentColumn.column);
          if (heatmap && heatmapOn && !heatmapSkipped.includes(currentColumn.column)) {
            columnDefs[currentColumn.column] = searcherMetricsValColumn(
              columnWidths[currentColumn.column],
              {
                max: heatmap.max,
                min: heatmap.min,
              },
            );
          } else {
            columnDefs[currentColumn.column] = searcherMetricsValColumn(
              columnWidths[currentColumn.column],
            );
          }
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [
    columnIds,
    columnDefs,
    projectColumnsMap,
    columnWidths,
    heatmapSkipped,
    projectHeatmap,
    heatmapOn,
  ]);

  const verticalBorder: DataEditorProps['verticalBorder'] = useCallback(
    (col: number) => !comparisonViewOpen && col === staticColumns.length + pinnedColumnsCount,
    [comparisonViewOpen, pinnedColumnsCount, staticColumns.length],
  );

  const sortMap = useMemo(() => {
    return sorts.reduce((acc, sort) => {
      if (sort.column && sort.direction) acc[sort.column] = sort.direction;
      return acc;
    }, {} as Record<string, string>);
  }, [sorts]);

  const drawHeader: DrawHeaderCallback = useCallback(
    ({ ctx, column, rect, theme }) => {
      if (!column.id || column.id === MULTISELECT) return false;

      const sortDirection = column.id && sortMap[column.id];
      if (sortDirection) {
        const arrowDirection = sortDirection === 'asc' ? 'up' : 'down';
        ctx.strokeStyle = theme.textLight;
        drawArrow(ctx, arrowDirection, rect.x + rect.width - 16, 12);
      }

      const xPad = theme.cellHorizontalPadding;
      const font = `${theme.baseFontStyle} ${theme.fontFamily}`;
      const middleCenterBias = getMiddleCenterBias(ctx, font);
      const x = rect.x + xPad;
      const y = rect.y + rect.height / 2 + middleCenterBias;
      const maxWidth = rect.width - (sortDirection ? 12 : 0) - 2 * theme.cellHorizontalPadding;

      ctx.fillStyle = theme.textHeader;
      drawTextWithEllipsis(ctx, column.title, x, y, maxWidth);

      return true;
    },
    [sortMap],
  );

  return (
    <div
      onWheel={() => {
        contextMenuOpen.set(false);
        closeTooltip();
      }}>
      {tooltip}
      <div className={css.base} ref={elementRef}>
        <DataEditor
          className={comparisonViewOpen ? css.compareTable : undefined}
          columns={columns}
          customRenderers={customRenderers}
          drawHeader={drawHeader}
          freezeColumns={isMobile ? 0 : staticColumns.length + pinnedColumnsCount}
          getCellContent={getCellContent}
          getCellsForSelection // `getCellsForSelection` is required for double click column resize to content.
          getRowThemeOverride={getRowThemeOverride}
          gridSelection={selection}
          headerHeight={36}
          headerIcons={headerIcons}
          height={height}
          minColumnWidth={MIN_COLUMN_WIDTH}
          ref={gridRef}
          rowHeight={rowHeightMap[rowHeight]}
          rows={dataTotal}
          smoothScrollX
          smoothScrollY
          theme={theme}
          verticalBorder={verticalBorder}
          width="100%"
          onCellClicked={onCellClicked}
          onCellContextMenu={onCellContextMenu}
          onColumnMoved={onColumnMoved}
          onColumnResize={handleColumnResize}
          onHeaderClicked={onHeaderClicked}
          onHeaderContextMenu={onHeaderClicked} // right-click
          onItemHovered={onColumnHovered}
          onVisibleRegionChanged={onScroll}
        />
      </div>
      <TableActionMenu {...menuProps} open={menuIsOpen} />
      {contextMenuProps && (
        <TableContextMenu
          {...contextMenuProps}
          open={contextMenuIsOpen}
          onComplete={handleContextMenuComplete}
        />
      )}
    </div>
  );
};

export default GlideTable;
