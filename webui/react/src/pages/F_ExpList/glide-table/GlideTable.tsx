import { FilterOutlined } from '@ant-design/icons';
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
import { v4 as uuidv4 } from 'uuid';

import { FilterFormStore, ROOT_ID } from 'components/FilterForm/components/FilterFormStore';
import {
  AvailableOperators,
  FormKind,
  Operator,
  SpecialColumnNames,
} from 'components/FilterForm/components/type';
import { handlePath } from 'routes/utils';
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import useUI from 'shared/contexts/stores/UI';
import usePrevious from 'shared/hooks/usePrevious';
import { AnyMouseEvent } from 'shared/utils/routes';
import usersStore from 'stores/users';
import { ExperimentWithTrial, Project, ProjectColumn } from 'types';
import { getProjectExperimentForExperimentItem } from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

import { PAGE_SIZE } from '../F_ExperimentList';
import { MapOfIdsToColors } from '../useGlasbey';

import {
  ColumnDef,
  defaultColumnWidths,
  defaultDateColumn,
  defaultNumberColumn,
  defaultTextColumn,
  ExperimentColumn,
  getColumnDefs,
  getHeaderIcons,
} from './columns';
import { TableContextMenu, TableContextMenuProps } from './contextMenu';
import { customRenderers } from './custom-renderers';
import { LinkCell } from './custom-renderers/cells/linkCell';
import css from './GlideTable.module.scss';
import { TableActionMenu, TableActionMenuProps } from './menu';
import { Sort, sortMenuItemsForColumn } from './MultiSortMenu';
import { BatchAction } from './TableActionBar';
import { useTableTooltip } from './tooltip';
import { getTheme } from './utils';

export interface GlideTableProps {
  clearSelectionTrigger?: number;
  colorMap: MapOfIdsToColors;
  excludedExperimentIds: Set<number>;
  data: Loadable<ExperimentWithTrial>[];
  dataTotal: number;
  handleScroll?: (r: Rectangle) => void;
  height: number;
  scrollPositionSetCount: WritableObservable<number>;
  sortableColumnIds: string[];
  setSortableColumnIds: (newColumns: string[]) => void;
  page: number;
  project?: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  selectedExperimentIds: number[];
  setExcludedExperimentIds: Dispatch<SetStateAction<Set<number>>>;
  setSelectedExperimentIds: Dispatch<SetStateAction<number[]>>;
  selectAll: boolean;
  setSelectAll: Dispatch<SetStateAction<boolean>>;
  handleUpdateExperimentList: (action: BatchAction, successfulIds: number[]) => void;
  sorts: Sort[];
  onSortChange: (sorts: Sort[]) => void;
  formStore: FilterFormStore;
  onIsOpenFilterChange: (value: boolean) => void;
  onContextMenuComplete?: () => void;
}

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

const isLinkCell = (cell: GridCell): cell is LinkCell => {
  return !!(cell as LinkCell).data?.link?.href;
};

export const GlideTable: React.FC<GlideTableProps> = ({
  data,
  dataTotal,
  excludedExperimentIds,
  clearSelectionTrigger,
  setSelectedExperimentIds,
  sortableColumnIds,
  setSortableColumnIds,
  colorMap,
  height,
  selectAll,
  setSelectAll,
  setExcludedExperimentIds,
  handleScroll,
  scrollPositionSetCount,
  page,
  project,
  handleUpdateExperimentList,
  onSortChange,
  sorts,
  projectColumns,
  formStore,
  onIsOpenFilterChange,
  onContextMenuComplete,
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

  const handleContextMenuComplete = useCallback(() => {
    onContextMenuComplete?.();
  }, [onContextMenuComplete]);

  const [contextMenuOpen] = useState(observable(false));
  const contextMenuIsOpen = useObservable(contextMenuOpen);

  const [contextMenuProps, setContextMenuProps] = useState<null | Omit<
    TableContextMenuProps,
    'open' | 'handleUpdateExperimentList'
  >>(null);

  const {
    ui: { theme: appTheme, darkLight },
  } = useUI();
  const theme = getTheme(appTheme);

  const users = useObservable(usersStore.getUsers());

  const columnIds = useMemo(() => [...STATIC_COLUMNS, ...sortableColumnIds], [sortableColumnIds]);

  const [selection, setSelection] = React.useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  });

  // Detect if user just click a row away from current selected group.
  // If this stand alone select is set, use it as the base when doing multi select.
  const [standAloneSelect, setStandAloneSelect] = React.useState<number>();

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
        .map((record) => record.data.experiment.id);
      if (prevIds === selectedIds) return prevIds;
      return selectedIds;
    });
  }, [selection.rows, setSelectedExperimentIds, data]);

  const [columnWidths, setColumnWidths] = useState<Record<string, number>>(defaultColumnWidths);

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
      if (!data[row]) return;
      const rowColorTheme = Loadable.match(data[row], {
        Loaded: (record) =>
          colorMap[record.experiment.id] ? { accentColor: colorMap[record.experiment.id] } : {},
        NotLoaded: () => ({}),
      });
      return { ...rowColorTheme };
    },
    [colorMap, data],
  );

  const onColumnResize: DataEditorProps['onColumnResize'] = useCallback(
    (column: GridColumn, width: number) => {
      const columnId = column.id;
      if (columnId === undefined || columnId === 'selected') return;
      setColumnWidths((prevWidths) => {
        if (columnId in prevWidths) {
          const prevWidth = prevWidths[columnId];
          if (width === prevWidth) return prevWidths;
        }
        return { ...prevWidths, [columnId]: width };
      });
    },
    [],
  );

  const deselectAllRows = useCallback(() => {
    setSelectAll(false);
    setSelection((prev) => ({ ...prev, rows: CompactSelection.empty() }));
  }, [setSelectAll, setSelection]);

  const selectAllRows = useCallback(() => {
    setExcludedExperimentIds(new Set());
    setSelectAll(true);
    setSelection(({ columns, rows }: GridSelection) => ({
      columns,
      rows: rows.add([0, data.length]),
    }));
  }, [setSelectAll, setSelection, data, setExcludedExperimentIds]);

  const previousData = usePrevious(data, undefined);
  useEffect(() => {
    if (selectAll && previousData && data.length > previousData.length) {
      setSelection(({ columns, rows }: GridSelection) => ({
        columns,
        rows: rows.add([previousData.length, data.length]),
      }));
    }
  }, [data, previousData, selectAll]);

  const onHeaderClicked: DataEditorProps['onHeaderClicked'] = React.useCallback(
    (col: number, args: HeaderClickedEventArgs) => {
      const columnId = columnIds[col];

      if (columnId === 'selected') {
        if (selectAll) {
          if (excludedExperimentIds.size) {
            selectAllRows();
          } else {
            deselectAllRows();
          }
        } else {
          if (selection.rows.length === data.length) {
            deselectAllRows();
          } else {
            selectAllRows();
          }
        }
        return;
      }
      const column = Loadable.getOrElse([], projectColumns).find((c) => c.column === columnId);
      if (!column) {
        return;
      }

      const filterMenuItemsForColumn = () => {
        const isSpecialColumn = (SpecialColumnNames as ReadonlyArray<string>).includes(
          column.column,
        );
        formStore.addChild(ROOT_ID, FormKind.Field, {
          index: formStore.formset.get().filterGroup.children.length,
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
        onIsOpenFilterChange(true);
        setMenuIsOpen(false);
      };

      const { bounds } = args;
      const items: MenuProps['items'] = [
        ...sortMenuItemsForColumn(column, sorts, onSortChange),
        { type: 'divider' },
        {
          icon: <FilterOutlined />,
          key: 'filter',
          label: 'Filter by this column',
          onClick: () => {
            setTimeout(() => {
              filterMenuItemsForColumn();
            }, 5);
          },
        },
      ];
      const x = bounds.x;
      const y = bounds.y + bounds.height;
      setMenuProps((prev) => ({ ...prev, items, title: `${columnId} menu`, x, y }));
      setMenuIsOpen(true);
    },
    [
      columnIds,
      projectColumns,
      sorts,
      onSortChange,
      selectAll,
      excludedExperimentIds.size,
      selectAllRows,
      deselectAllRows,
      selection.rows.length,
      data.length,
      formStore,
      onIsOpenFilterChange,
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
        Loaded: (rowData) => {
          const columnId = columnIds[col];
          return columnDefs[columnId].renderer(rowData, row);
        },
        NotLoaded: () => loadingCell,
      });
    },
    [appTheme, data, columnIds, columnDefs],
  );

  const onCellClicked: DataEditorProps['onCellClicked'] = useCallback(
    (cell: Item, event: CellClickedEventArgs) => {
      const findConsecutiveBefore = (rows: number[], row: number) => {
        while (row >= 0) {
          row = row - 1;
          if (!rows.includes(row)) return row + 1;
        }
        return row;
      };
      const findConsecutiveAfter = (rows: number[], row: number) => {
        while (row < data.length) {
          row = row + 1;
          if (!rows.includes(row)) return row - 1;
        }
        return row;
      };
      const isStandAlone = (rows: CompactSelection, row: number) => {
        if (row === 0) return !rows.hasIndex(row + 1);
        if (row === data.length - 1) return !rows.hasIndex(row - 1);
        return !rows.hasIndex(row - 1) && !rows.hasIndex(row + 1);
      };

      setStandAloneSelect(undefined);

      const [col, row] = cell;
      Loadable.match(data[row], {
        Loaded: (rowData) => {
          const columnId = columnIds[col];
          const cell = columnDefs[columnId].renderer(rowData, row);

          if (isLinkCell(cell)) {
            handlePath(event as unknown as AnyMouseEvent, { path: cell.data.link.href });
            // cell.data.link.onClick(event);
          } else {
            if (event.shiftKey) {
              setSelection(({ rows }: GridSelection) => {
                if (standAloneSelect && standAloneSelect > row) {
                  return {
                    columns: CompactSelection.empty(),
                    rows: event.metaKey
                      ? rows.add([row, standAloneSelect + 1])
                      : CompactSelection.fromSingleSelection([row, standAloneSelect + 1]),
                  };
                }
                const rowsArray = rows.toArray();
                const smallestClosest = rowsArray.filter((r) => r < row).last();
                const largestClosest = rowsArray.filter((r) => r > row).first();
                const smallestLinked = findConsecutiveBefore(rowsArray, smallestClosest);
                const greatestLinked = findConsecutiveAfter(rowsArray, largestClosest);
                return {
                  columns: CompactSelection.empty(),
                  rows:
                    smallestClosest >= 0
                      ? event.metaKey
                        ? rows.add([smallestClosest, row + 1])
                        : CompactSelection.fromSingleSelection([smallestLinked, row + 1])
                      : largestClosest
                      ? event.metaKey
                        ? rows.add([row, largestClosest + 1])
                        : CompactSelection.fromSingleSelection([row, greatestLinked + 1])
                      : CompactSelection.fromSingleSelection(row),
                };
              });
            } else {
              isStandAlone(selection.rows, row) &&
                !selection.rows.hasIndex(row) &&
                setStandAloneSelect(row);

              if (selection.rows.hasIndex(row)) {
                setSelection(({ columns, rows }: GridSelection) => ({
                  columns,
                  rows: rows.remove(row),
                }));
                if (selectAll) {
                  const experiment = data[row];
                  if (Loadable.isLoaded(experiment)) {
                    setExcludedExperimentIds((prev) => {
                      if (experiment.data.experiment) {
                        return new Set([...prev, experiment.data.experiment?.id]);
                      } else {
                        return prev;
                      }
                    });
                  }
                }
              } else {
                setSelection(({ columns, rows }: GridSelection) => ({
                  columns,
                  rows: rows.add(row),
                }));
                const experiment = data[row];
                if (Loadable.isLoaded(experiment)) {
                  setExcludedExperimentIds((prev) => {
                    return new Set([...prev].filter((id) => id !== experiment.data.experiment?.id));
                  });
                }
              }
            }
          }
        },
        NotLoaded: () => null,
      });
    },
    [data, columnIds, columnDefs, selection, selectAll, setExcludedExperimentIds, standAloneSelect],
  );

  const onCellContextMenu: DataEditorProps['onCellContextMenu'] = useCallback(
    (cell: Item, event: CellClickedEventArgs) => {
      // Close existing context menu.
      contextMenuOpen.set(false);

      const [col, row] = cell;
      Loadable.match(data[row], {
        Loaded: (rowData) => {
          // Prevent the browser native context menu from showing up.
          event.preventDefault();

          // Delay needed due to the call to close previously existing context menu.
          setTimeout(() => {
            const columnId = columnIds[col];
            const cell = columnDefs[columnId].renderer(rowData, row);

            // Update the context menu based on the cell context.
            setContextMenuProps({
              experiment: getProjectExperimentForExperimentItem(rowData.experiment, project),
              handleClose: (e?: Event) => {
                // Prevent the context menu closing click from triggering something else.
                if (contextMenuOpen.get()) e?.stopPropagation();
                contextMenuOpen.set(false);
              },
              link: isLinkCell(cell) ? cell.data.link.href : undefined,
              x: Math.max(0, event.bounds.x + event.localEventX - 4),
              y: Math.max(0, event.bounds.y + event.localEventY - 4),
            });

            contextMenuOpen.set(true);
          }, 50);
        },
        NotLoaded: () => null,
      });
    },
    [columnDefs, columnIds, data, project, setContextMenuProps, contextMenuOpen],
  );

  const onColumnMoved: DataEditorProps['onColumnMoved'] = useCallback(
    (columnIdsStartIdx: number, columnIdsEndIdx: number): void => {
      const sortableColumnIdsStartIdx = columnIdsStartIdx - STATIC_COLUMNS.length;
      const sortableColumnIdsEndIdx = Math.max(columnIdsEndIdx - STATIC_COLUMNS.length, 0);
      if (sortableColumnIdsStartIdx > -1) {
        const newCols = [...sortableColumnIds];
        const [toMove] = newCols.splice(sortableColumnIdsStartIdx, 1);
        newCols.splice(sortableColumnIdsEndIdx, 0, toMove);
        setSortableColumnIds(newCols);
      }
    },
    [sortableColumnIds, setSortableColumnIds],
  );

  const columns: DataEditorProps['columns'] = useMemo(() => {
    const gridColumns = columnIds
      .map((columnName) => {
        if (columnName in columnDefs) return columnDefs[columnName];
        if (!Loadable.isLoaded(projectColumnsMap)) return;
        const currentColumn = projectColumnsMap.data[columnName];
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
            dataPath = `bestTrial.bestValidationMetric.metrics.${currentColumn.column.replace(
              'validation.',
              '',
            )}`;
            break;
          case V1LocationType.UNSPECIFIED:
          default:
            break;
        }
        switch (currentColumn.type) {
          case V1ColumnType.NUMBER:
            columnDefs[currentColumn.column] = defaultNumberColumn(
              currentColumn,
              columnWidths,
              dataPath,
            );
            break;
          case V1ColumnType.DATE:
            columnDefs[currentColumn.column] = defaultDateColumn(
              currentColumn,
              columnWidths,
              dataPath,
            );
            break;
          case V1ColumnType.TEXT:
          case V1ColumnType.UNSPECIFIED:
          default:
            columnDefs[currentColumn.column] = defaultTextColumn(
              currentColumn,
              columnWidths,
              dataPath,
            );
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [columnIds, columnDefs, projectColumnsMap, columnWidths]);

  const verticalBorder: DataEditorProps['verticalBorder'] = useCallback(
    (col: number) => columnIds[col - 1] === STATIC_COLUMNS.last(),
    [columnIds],
  );

  return (
    <div
      onWheel={() => {
        contextMenuOpen.set(false);
        closeTooltip();
      }}>
      {tooltip}
      <div className={css.base}>
        <DataEditor
          columns={columns}
          customRenderers={customRenderers}
          freezeColumns={STATIC_COLUMNS.length}
          getCellContent={getCellContent}
          // `getCellsForSelection` is required for double click column resize to content.
          getCellsForSelection
          getRowThemeOverride={getRowThemeOverride}
          gridSelection={selection}
          headerHeight={36}
          headerIcons={headerIcons}
          height={height}
          ref={gridRef}
          rowHeight={40}
          rows={dataTotal}
          smoothScrollX
          smoothScrollY
          theme={theme}
          verticalBorder={verticalBorder}
          width="100%"
          onCellClicked={onCellClicked}
          onCellContextMenu={onCellContextMenu}
          onColumnMoved={onColumnMoved}
          onColumnResize={onColumnResize}
          onHeaderClicked={onHeaderClicked}
          onItemHovered={onItemHovered}
          onVisibleRegionChanged={handleScroll}
        />
      </div>
      <TableActionMenu {...menuProps} open={menuIsOpen} />
      {contextMenuProps && (
        <TableContextMenu
          {...contextMenuProps}
          handleUpdateExperimentList={handleUpdateExperimentList}
          open={contextMenuIsOpen}
          onComplete={handleContextMenuComplete}
        />
      )}
    </div>
  );
};

export default GlideTable;
