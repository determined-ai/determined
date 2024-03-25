import { CompactSelection, GridSelection } from '@glideapps/glide-data-grid';
import { isLeft } from 'fp-ts/lib/Either';
import Column from 'hew/Column';
import {
  ColumnDef,
  defaultDateColumn,
  defaultNumberColumn,
  defaultSelectionColumn,
  defaultTextColumn,
  MULTISELECT,
} from 'hew/DataGrid/columns';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
import DataGrid, { Sort, validSort, ValidSort } from 'hew/DataGrid/DataGrid';
import Link from 'hew/Link';
import Message from 'hew/Message';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import { IOFilterFormSet } from 'components/FilterForm/components/type';
import useUI from 'components/ThemeProvider';
import { useGlasbey } from 'hooks/useGlasbey';
import useMobile from 'hooks/useMobile';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { Error } from 'pages/F_ExpList/glide-table/exceptions';
import { EMPTY_SORT } from 'pages/F_ExpList/glide-table/MultiSortMenu';
import { paths } from 'routes/utils';
import { getProjectColumns, searchRuns } from 'services/api';
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import usersStore from 'stores/users';
import { ExperimentAction, FlatRun, Project, ProjectColumn } from 'types';
import handleError from 'utils/error';

import { getColumnDefs, RunColumn, runColumns } from './columns';
import css from './FlatRuns.module.scss';
import {
  FlatRunsGlobalSettings,
  FlatRunsSettings,
  settingsConfigForProject,
  settingsConfigGlobal,
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
  const contentRef = useRef<HTMLDivElement>(null);
  const [searchParams, setSearchParams] = useSearchParams();
  const settingsConfig = useMemo(() => settingsConfigForProject(project.id), [project.id]);
  const {
    isLoading: isLoadingSettings,
    settings,
    updateSettings,
  } = useSettings<FlatRunsSettings>(settingsConfig);
  const { settings: globalSettings } = useSettings<FlatRunsGlobalSettings>(settingsConfigGlobal);

  const [runs, setRuns] = useState<Loadable<FlatRun>[]>(INITIAL_LOADING_RUNS);
  const isPagedView = globalSettings.tableViewMode === 'paged';
  const [page, setPage] = useState(() =>
    isFinite(Number(searchParams.get('page'))) ? Math.max(Number(searchParams.get('page')), 0) : 0,
  );
  const [projectColumns, setProjectColumns] = useState<Loadable<ProjectColumn[]>>(NotLoaded);

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
  const users = useObservable(usersStore.getUsers());

  const colorMap = useGlasbey(settings.selectedRuns);

  const {
    ui: { theme: appTheme },
    isDarkMode,
  } = useUI();

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

  const [selection, setSelection] = React.useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  });

  const selectAll = useMemo<boolean>(
    () => !isLoadingSettings && settings.selectAll,
    [isLoadingSettings, settings.selectAll],
  );

  const selectedRunIds: Set<number> = useMemo(() => {
    return isLoadingSettings ? new Set() : new Set(settings.selectedRuns);
  }, [isLoadingSettings, settings.selectedRuns]);

  const excludedRunIds: Set<number> = useMemo(() => {
    return isLoadingSettings ? new Set() : new Set(settings.excludedRuns);
  }, [isLoadingSettings, settings.excludedRuns]);

  useEffect(() => {
    if (isLoading) return;

    const selectedIds = new Set(selectedRunIds);
    const loadedRuns = Loadable.filterNotLoaded(runs);

    if (selectAll) {
      loadedRuns.forEach((run) => {
        const id = run.id;
        if (!excludedRunIds.has(id)) selectedIds.add(id);
      });
      updateSettings({ selectedRuns: Array.from(selectedIds) });
    }

    /**
     * Use settings info (selectionAll, selectedRunIds, excludedRunIds)
     * to figure out and update list selections.
     */
    setSelection((prevSelection) => {
      let rows = CompactSelection.empty();
      loadedRuns.forEach((run, index) => {
        const id = run.id;
        if ((selectAll && !excludedRunIds.has(id)) || (!selectAll && selectedIds.has(id))) {
          rows = rows.add(index);
        }
      });
      return { ...prevSelection, rows };
    });
  }, [excludedRunIds, isLoading, runs, selectAll, selectedRunIds, total, updateSettings]);

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
      rowSelection: selection.rows,
      selectAll,
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
          return (columnDefs[columnName] = defaultSelectionColumn(selection.rows, selectAll));
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
          case V1ColumnType.NUMBER: {
            columnDefs[currentColumn.column] = defaultNumberColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column],
              dataPath,
            );
            // }
            break;
          }
          case V1ColumnType.DATE:
            columnDefs[currentColumn.column] = defaultDateColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column],
              dataPath,
            );
            break;
          case V1ColumnType.TEXT:
          case V1ColumnType.UNSPECIFIED:
          default:
            columnDefs[currentColumn.column] = defaultTextColumn(
              currentColumn.column,
              currentColumn.displayName || currentColumn.column,
              settings.columnWidths[currentColumn.column],
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
    selectAll,
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
          ...INITIAL_LOADING_RUNS,
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
    setSelection({ columns: CompactSelection.empty(), rows: CompactSelection.empty() });
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
      updateSettings({ columnWidths: { ...settings.columnWidths, [columnId]: width } });
    },
    [settings.columnWidths, updateSettings],
  );

  const handleContextMenuComplete: ContextMenuCompleteHandlerProps<ExperimentAction, FlatRun> =
    useCallback(() => {}, []);

  const handleColumnsOrderChange = useCallback(
    (newColumnsOrder: string[]) => {
      updateSettings({ columns: newColumnsOrder });
    },
    [updateSettings],
  );

  const getRowAccentColor = (rowData: FlatRun) => {
    return colorMap[rowData.id];
  };

  // TODO: poll?
  useEffect(() => {
    (async () => {
      try {
        const columns = await getProjectColumns({ id: project.id });
        columns.sort((a, b) =>
          a.location === V1LocationType.EXPERIMENT && b.location === V1LocationType.EXPERIMENT
            ? runColumns.indexOf(a.column as RunColumn) - runColumns.indexOf(b.column as RunColumn)
            : 0,
        );
        setProjectColumns(Loaded(columns));
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch project columns' });
      }
    })();
  }, [project.id]);

  useEffect(
    () =>
      formStore.asJsonString.subscribe(() => {
        resetPagination();
        const loadableFormset = formStore.formset.get();
        Loadable.forEach(loadableFormset, (formSet) =>
          updateSettings({ filterset: JSON.stringify(formSet) }),
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
            getRowAccentColor={getRowAccentColor}
            isPaginated={isPagedView}
            page={page}
            pageSize={PAGE_SIZE}
            selection={selection}
            staticColumns={STATIC_COLUMNS}
            total={isPagedView ? runs.length : Loadable.getOrElse(PAGE_SIZE, total)}
            onColumnResize={handleColumnWidthChange}
            onColumnsOrderChange={handleColumnsOrderChange}
            onContextMenuComplete={handleContextMenuComplete}
            onPageUpdate={handlePageUpdate}
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
