import { CompactSelection, GridSelection } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import Column from 'hew/Column';
import {
  ColumnDef,
  defaultDateColumn,
  defaultNumberColumn,
  defaultSelectionColumn,
  defaultTextColumn,
  MIN_COLUMN_WIDTH,
  MULTISELECT,
} from 'hew/DataGrid/columns';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
import DataGrid, {
  DataGridHandle,
  HandleSelectionChangeType,
  RangelessSelectionType,
  SelectionType,
  Sort,
  validSort,
  ValidSort,
} from 'hew/DataGrid/DataGrid';
import { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Link from 'hew/Link';
import Message from 'hew/Message';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { Error } from 'components/exceptions';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import { IOFilterFormSet } from 'components/FilterForm/components/type';
import { EMPTY_SORT, sortMenuItemsForColumn } from 'components/MultiSortMenu';
import { RowHeight } from 'components/OptionsMenu';
import {
  DataGridGlobalSettings,
  rowHeightMap,
  settingsConfigGlobal,
} from 'components/OptionsMenu.settings';
import useUI from 'components/ThemeProvider';
import { useAsync } from 'hooks/useAsync';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import {
  DEFAULT_SELECTION,
  SelectionType as SelectionState,
} from 'pages/F_ExpList/F_ExperimentList.settings';
import { paths } from 'routes/utils';
import { getProjectColumns, searchRuns } from 'services/api';
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import { DetailedUser, ExperimentAction, FlatRun, Project, ProjectColumn } from 'types';
import handleError from 'utils/error';

import { defaultColumnWidths, getColumnDefs, RunColumn, runColumns } from './columns';
import css from './FlatRuns.module.scss';
import {
  defaultFlatRunsSettings,
  FlatRunsSettings,
  settingsPathForProject,
} from './FlatRuns.settings';

export const PAGE_SIZE = 100;
const INITIAL_LOADING_RUNS: Loadable<FlatRun>[] = new Array(PAGE_SIZE).fill(NotLoaded);

const STATIC_COLUMNS = [MULTISELECT];

const formStore = new FilterFormStore();

interface Props {
  project: Project;
}

const makeSortString = (sorts: ValidSort[]): string =>
  sorts.map((s) => `${s.column}=${s.direction}`).join(',');

const parseSortString = (sortString: string): Sort[] => {
  if (!sortString) return [EMPTY_SORT];
  const components = sortString.split(',');
  return components.map((c) => {
    const [column, direction] = c.split('=', 2);
    return {
      column,
      direction: direction === 'asc' || direction === 'desc' ? direction : undefined,
    };
  });
};

const FlatRuns: React.FC<Props> = ({ project }) => {
  const dataGridRef = useRef<DataGridHandle>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [searchParams, setSearchParams] = useSearchParams();

  const settingsPath = useMemo(() => settingsPathForProject(project.id), [project.id]);
  const flatRunsSettingsObs = useMemo(
    () => userSettings.get(FlatRunsSettings, settingsPath),
    [settingsPath],
  );
  const flatRunsSettings = useObservable(flatRunsSettingsObs);
  const isLoadingSettings = useMemo(() => flatRunsSettings.isNotLoaded, [flatRunsSettings]);
  const updateSettings = useCallback(
    (p: Partial<FlatRunsSettings>) => userSettings.setPartial(FlatRunsSettings, settingsPath, p),
    [settingsPath],
  );
  const settings = useMemo(
    () =>
      flatRunsSettings
        .map((s) => ({ ...defaultFlatRunsSettings, ...s }))
        .getOrElse(defaultFlatRunsSettings),
    [flatRunsSettings],
  );

  const { settings: globalSettings } = useSettings<DataGridGlobalSettings>(settingsConfigGlobal);

  const [runs, setRuns] = useState<Loadable<FlatRun>[]>(INITIAL_LOADING_RUNS);
  const isPagedView = true;
  const [page, setPage] = useState(() =>
    isFinite(Number(searchParams.get('page'))) ? Math.max(Number(searchParams.get('page')), 0) : 0,
  );

  const [sorts, setSorts] = useState<Sort[]>(() => {
    if (!isLoadingSettings) {
      return parseSortString(settings.sortString);
    }
    return [EMPTY_SORT];
  });
  const sortString = useMemo(() => makeSortString(sorts.filter(validSort.is)), [sorts]);
  const loadableFormset = useObservable(formStore.formset);
  const [total, setTotal] = useState<Loadable<number>>(NotLoaded);
  const isMobile = useMobile();
  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());
  const users = useObservable<Loadable<DetailedUser[]>>(userStore.getUsers());

  const {
    ui: { theme: appTheme },
    isDarkMode,
  } = useUI();

  const projectColumns = useAsync(async () => {
    try {
      const columns = await getProjectColumns({ id: project.id });
      columns.sort((a, b) =>
        a.location === V1LocationType.EXPERIMENT && b.location === V1LocationType.EXPERIMENT
          ? runColumns.indexOf(a.column as RunColumn) - runColumns.indexOf(b.column as RunColumn)
          : 0,
      );
      return columns;
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch project columns' });
      return NotLoaded;
    }
  }, [project.id]);

  const columnsIfLoaded = useMemo(
    () => (isLoadingSettings ? [] : settings.columns),
    [isLoadingSettings, settings.columns],
  );

  const showPagination = useMemo(() => {
    return (
      isPagedView &&
      (!settings.compare || settings.pinnedColumnsCount !== 0) &&
      !(isMobile && settings.compare)
    );
  }, [isMobile, isPagedView, settings.compare, settings.pinnedColumnsCount]);

  const loadedSelectedRunIds = useMemo(() => {
    const selectedMap = new Map<number, { run: FlatRun; index: number }>();
    if (isLoadingSettings) {
      return selectedMap;
    }
    const selectedIdSet = new Set(
      settings.selection.type === 'ONLY_IN' ? settings.selection.selections : [],
    );
    runs.forEach((r, index) => {
      Loadable.forEach(r, (run) => {
        if (selectedIdSet.has(run.id)) {
          selectedMap.set(run.id, { index, run });
        }
      });
    });
    return selectedMap;
  }, [isLoadingSettings, settings.selection, runs]);

  const selection = useMemo<GridSelection>(() => {
    let rows = CompactSelection.empty();
    loadedSelectedRunIds.forEach((info) => {
      rows = rows.add(info.index);
    });
    return {
      columns: CompactSelection.empty(),
      rows,
    };
  }, [loadedSelectedRunIds]);

  const colorMap = useGlasbey([...loadedSelectedRunIds.keys()]);

  const columns: ColumnDef<FlatRun>[] = useMemo(() => {
    const projectColumnsMap: Loadable<Record<string, ProjectColumn>> = Loadable.map(
      projectColumns,
      (columns) => {
        return columns.reduce((acc, col) => ({ ...acc, [col.column]: col }), {});
      },
    );
    const columnDefs = getColumnDefs({
      appTheme,
      columnWidths: settings.columnWidths,
      themeIsDark: isDarkMode,
      users,
    });
    const gridColumns = (
      settings.compare
        ? [...STATIC_COLUMNS, ...columnsIfLoaded.slice(0, settings.pinnedColumnsCount)]
        : [...STATIC_COLUMNS, ...columnsIfLoaded]
    )
      .map((columnName) => {
        if (columnName === MULTISELECT) {
          return defaultSelectionColumn(selection.rows, false);
        }

        if (columnName in columnDefs) return columnDefs[columnName];
        const currentColumn = projectColumnsMap.getOrElse({})[columnName];
        if (!currentColumn) return;
        let dataPath: string | undefined = undefined;

        switch (currentColumn.location) {
          case V1LocationType.EXPERIMENT:
            dataPath = `experiment.${currentColumn.column}`;
            break;
          case V1LocationType.RUN:
            dataPath = currentColumn.column;
            break;
          case V1LocationType.HYPERPARAMETERS:
          case V1LocationType.RUNHYPERPARAMETERS:
            dataPath = `hyperparameters.${currentColumn.column.replace('hp.', '')}.val`;
            break;
          case V1LocationType.VALIDATIONS:
            dataPath = `summaryMetrics.validationMetrics.${currentColumn.column.replace(
              'validation.',
              '',
            )}`;
            break;
          case V1LocationType.TRAINING:
            dataPath = `summaryMetrics.avgMetrics.${currentColumn.column.replace('training.', '')}`;
            break;
          case V1LocationType.CUSTOMMETRIC:
            dataPath = `summaryMetrics.${currentColumn.column}`;
            break;
          case V1LocationType.UNSPECIFIED:
          default:
            break;
        }
        switch (currentColumn.type) {
          case V1ColumnType.NUMBER:
            columnDefs[currentColumn.column] = defaultNumberColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column] ??
                defaultColumnWidths[currentColumn.column as RunColumn] ??
                MIN_COLUMN_WIDTH,
              dataPath,
            );
            break;
          case V1ColumnType.DATE:
            columnDefs[currentColumn.column] = defaultDateColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column] ??
                defaultColumnWidths[currentColumn.column as RunColumn] ??
                MIN_COLUMN_WIDTH,
              dataPath,
            );
            break;
          case V1ColumnType.TEXT:
          case V1ColumnType.UNSPECIFIED:
          default:
            columnDefs[currentColumn.column] = defaultTextColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column] ??
                defaultColumnWidths[currentColumn.column as RunColumn] ??
                MIN_COLUMN_WIDTH,
              dataPath,
            );
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [
    appTheme,
    columnsIfLoaded,
    isDarkMode,
    projectColumns,
    selection.rows,
    settings.columnWidths,
    settings.compare,
    settings.pinnedColumnsCount,
    users,
  ]);

  const onPageChange = useCallback(
    (cPage: number, cPageSize: number) => {
      updateSettings({ pageLimit: cPageSize });
      // Pagination component is assuming starting index of 1.
      setPage((prevPage) => {
        if (cPage - 1 !== prevPage) {
          setRuns(Array(cPageSize).fill(NotLoaded));
        }
        return cPage - 1;
      });
    },
    [updateSettings],
  );

  const fetchRuns = useCallback(async (): Promise<void> => {
    if (isLoadingSettings || Loadable.isNotLoaded(loadableFormset)) return;
    try {
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);
      const response = await searchRuns(
        {
          //filter: filtersString,
          limit: isPagedView ? settings.pageLimit : 2 * PAGE_SIZE,
          offset: isPagedView ? page * settings.pageLimit : tableOffset,
          projectId: project.id,
          sort: sortString || undefined,
        },
        { signal: canceler.signal },
      );
      const loadedRuns = response.runs;

      setRuns((prev) => {
        if (isPagedView) {
          return loadedRuns.map((run) => Loaded(run));
        }

        // Update the list with the fetched results.
        return prev.toSpliced(
          tableOffset,
          loadedRuns.length,
          ...loadedRuns.map((experiment) => Loaded(experiment)),
        );
      });
      setTotal(
        response.pagination.total !== undefined ? Loaded(response.pagination.total) : NotLoaded,
      );
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch runs.' });
    } finally {
      setIsLoading(false);
    }
  }, [
    canceler.signal,
    isLoadingSettings,
    isPagedView,
    loadableFormset,
    page,
    project.id,
    settings.pageLimit,
    sortString,
  ]);

  const { stopPolling } = usePolling(fetchRuns, { rerunOnNewFn: true });

  const handlePageUpdate = useCallback((page: number) => {
    setPage(page);
  }, []);

  const numFilters = 0;

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setRuns(INITIAL_LOADING_RUNS);
  }, []);

  useEffect(() => {
    if (!isLoadingSettings && settings.sortString) {
      setSorts(parseSortString(settings.sortString));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoadingSettings]);

  useEffect(() => {
    setSearchParams((params) => {
      if (page) {
        params.set('page', page.toString());
      } else {
        params.delete('page');
      }
      return params;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  useEffect(() => {
    // useSettings load the default value first, and then load the data from DB
    // use this useEffect to re-init the correct useSettings value when settings.filterset is changed
    if (isLoadingSettings) return;
    const formSetValidation = IOFilterFormSet.decode(JSON.parse(settings.filterset));
    if (isLeft(formSetValidation)) {
      handleError(formSetValidation.left, {
        publicSubject: 'Unable to initialize filterset from settings',
      });
    } else {
      const formset = formSetValidation.right;
      formStore.init(formset);
    }
  }, [settings.filterset, isLoadingSettings]);

  const handleColumnWidthChange = useCallback(
    (columnId: string, width: number) => {
      updateSettings({
        columnWidths: { ...settings.columnWidths, [columnId]: Math.max(MIN_COLUMN_WIDTH, width) },
      });
    },
    [settings.columnWidths, updateSettings],
  );

  const rowRangeToIds = useCallback(
    (range: [number, number]) => {
      const slice = runs.slice(range[0], range[1]);
      return Loadable.filterNotLoaded(slice).map((run) => run.id);
    },
    [runs],
  );

  const handleSelectionChange: HandleSelectionChangeType = useCallback(
    (selectionType: SelectionType | RangelessSelectionType, range?: [number, number]) => {
      let newSettings: SelectionState = { ...settings.selection };

      switch (selectionType) {
        case 'add':
          if (!range) return;
          if (newSettings.type === 'ALL_EXCEPT') {
            const excludedSet = new Set(newSettings.exclusions);
            rowRangeToIds(range).forEach((id) => excludedSet.delete(id));
            newSettings.exclusions = Array.from(excludedSet);
          } else {
            const includedSet = new Set(newSettings.selections);
            rowRangeToIds(range).forEach((id) => includedSet.add(id));
            newSettings.selections = Array.from(includedSet);
          }

          break;
        case 'add-all':
          newSettings = {
            exclusions: [],
            type: 'ALL_EXCEPT' as const,
          };

          break;
        case 'remove':
          if (!range) return;
          if (newSettings.type === 'ALL_EXCEPT') {
            const excludedSet = new Set(newSettings.exclusions);
            rowRangeToIds(range).forEach((id) => excludedSet.add(id));
            newSettings.exclusions = Array.from(excludedSet);
          } else {
            const includedSet = new Set(newSettings.selections);
            rowRangeToIds(range).forEach((id) => includedSet.delete(id));
            newSettings.selections = Array.from(includedSet);
          }

          break;
        case 'remove-all':
          newSettings = DEFAULT_SELECTION;

          break;
        case 'set':
          if (!range) return;
          newSettings = {
            ...DEFAULT_SELECTION,
            selections: Array.from(rowRangeToIds(range)),
          };

          break;
      }

      updateSettings({ selection: newSettings });
    },
    [rowRangeToIds, settings.selection, updateSettings],
  );

  const handleContextMenuComplete: ContextMenuCompleteHandlerProps<ExperimentAction, FlatRun> =
    useCallback(() => {}, []);

  const handleColumnsOrderChange = useCallback(
    (newColumnsOrder: string[]) => {
      updateSettings({ columns: newColumnsOrder });
    },
    [updateSettings],
  );

  const handleSortChange = useCallback(
    (sorts: Sort[]) => {
      setSorts(sorts);
      const newSortString = makeSortString(sorts.filter(validSort.is));
      if (newSortString !== sortString) {
        resetPagination();
      }
      updateSettings({ sortString: newSortString });
    },
    [resetPagination, sortString, updateSettings],
  );

  const getRowAccentColor = (rowData: FlatRun) => {
    return colorMap[rowData.id];
  };

  const handlePinnedColumnsCountChange = useCallback(
    (newCount: number) => updateSettings({ pinnedColumnsCount: newCount }),
    [updateSettings],
  );

  const getHeaderMenuItems = useCallback(
    (columnId: string, colIdx: number): MenuItem[] => {
      if (columnId === MULTISELECT) {
        const items: MenuItem[] = [
          selection.rows.length > 0
            ? {
                key: 'select-none',
                label: 'Clear selected',
                onClick: () => {
                  handleSelectionChange?.('remove-all');
                },
              }
            : null,
          ...[5, 10, 25].map((n) => ({
            key: `select-${n}`,
            label: `Select first ${n}`,
            onClick: () => {
              handleSelectionChange?.('set', [0, n]);
              dataGridRef.current?.scrollToTop();
            },
          })),
          {
            key: 'select-all',
            label: 'Select all',
            onClick: () => {
              handleSelectionChange?.('set', [0, settings.pageLimit]);
            },
          },
        ];
        return items;
      }

      const isPinned = colIdx <= settings.pinnedColumnsCount + STATIC_COLUMNS.length - 1;
      const items: MenuItem[] = [
        // Column is pinned if the index is inside of the frozen columns
        colIdx < STATIC_COLUMNS.length || isMobile
          ? null
          : !isPinned
            ? {
                icon: <Icon decorative name="pin" />,
                key: 'pin',
                label: 'Pin column',
                onClick: () => {
                  const newColumnsOrder = columnsIfLoaded.filter((c) => c !== columnId);
                  newColumnsOrder.splice(settings.pinnedColumnsCount, 0, columnId);
                  handleColumnsOrderChange?.(newColumnsOrder);
                  handlePinnedColumnsCountChange?.(
                    Math.min(settings.pinnedColumnsCount + 1, columnsIfLoaded.length),
                  );
                },
              }
            : {
                disabled: settings.pinnedColumnsCount <= 1,
                icon: <Icon decorative name="pin" />,
                key: 'unpin',
                label: 'Unpin column',
                onClick: () => {
                  const newColumnsOrder = columnsIfLoaded.filter((c) => c !== columnId);
                  newColumnsOrder.splice(settings.pinnedColumnsCount - 1, 0, columnId);
                  handleColumnsOrderChange?.(newColumnsOrder);
                  handlePinnedColumnsCountChange?.(Math.max(settings.pinnedColumnsCount - 1, 0));
                },
              },
      ];
      const column = Loadable.getOrElse([], projectColumns).find((c) => c.column === columnId);
      if (!column) return items;

      const BANNED_FILTER_COLUMNS = ['searcherMetricsVal'];
      const sortOptions = sortMenuItemsForColumn(column, sorts, handleSortChange);
      if (sortOptions.length > 0) {
        items.push(
          ...(BANNED_FILTER_COLUMNS.includes(column.column)
            ? []
            : [
                { type: 'divider' as const },
                ...sortMenuItemsForColumn(column, sorts, handleSortChange),
              ]),
        );
      }
      return items;
    },
    [
      columnsIfLoaded,
      handleColumnsOrderChange,
      handlePinnedColumnsCountChange,
      handleSelectionChange,
      handleSortChange,
      isMobile,
      projectColumns,
      selection.rows.length,
      settings.pinnedColumnsCount,
      sorts,
      settings.pageLimit,
    ],
  );

  useEffect(
    () =>
      formStore.asJsonString.subscribe(() => {
        resetPagination();
        const loadableFormset = formStore.formset.get();
        Loadable.forEach(loadableFormset, (formSet) =>
          updateSettings({
            filterset: JSON.stringify(formSet),
            selection: DEFAULT_SELECTION,
          }),
        );
      }),
    [resetPagination, updateSettings],
  );

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  return (
    <div className={css.content} ref={contentRef}>
      {!isLoading && total.isLoaded && total.data === 0 ? (
        numFilters === 0 ? (
          <Message
            action={
              <Link external href={paths.docs('/get-started/webui-qs.html')}>
                Quick Start Guide
              </Link>
            }
            description="Keep track of runs in a project by connecting up your code."
            icon="experiment"
            title="No Runs"
          />
        ) : (
          <Message description="No results matching your filters" icon="search" />
        )
      ) : error ? (
        <Error fetchData={fetchRuns} />
      ) : (
        <>
          <DataGrid
            columns={columns}
            data={runs}
            getHeaderMenuItems={getHeaderMenuItems}
            getRowAccentColor={getRowAccentColor}
            imperativeRef={dataGridRef}
            isPaginated={isPagedView}
            page={page}
            pageSize={PAGE_SIZE}
            pinnedColumnsCount={isLoadingSettings ? 0 : settings.pinnedColumnsCount}
            rowHeight={rowHeightMap[globalSettings.rowHeight as RowHeight]}
            selection={selection}
            sorts={sorts}
            staticColumns={STATIC_COLUMNS}
            total={total.getOrElse(PAGE_SIZE)}
            onColumnResize={handleColumnWidthChange}
            onColumnsOrderChange={handleColumnsOrderChange}
            onContextMenuComplete={handleContextMenuComplete}
            onPageUpdate={handlePageUpdate}
            onPinnedColumnsCountChange={handlePinnedColumnsCountChange}
            onSelectionChange={handleSelectionChange}
          />
          {showPagination && (
            <Row>
              <Column align="right">
                <Pagination
                  current={page + 1}
                  pageSize={settings.pageLimit}
                  pageSizeOptions={[20, 40, 80]}
                  total={Loadable.getOrElse(0, total)}
                  onChange={onPageChange}
                />
              </Column>
            </Row>
          )}
        </>
      )}
    </div>
  );
};

export default FlatRuns;
