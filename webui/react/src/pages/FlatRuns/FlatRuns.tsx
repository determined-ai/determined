import { CompactSelection, GridSelection } from '@glideapps/glide-data-grid';
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
import DataGrid, { Sort, ValidSort } from 'hew/DataGrid/DataGrid';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { observable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import useUI from 'components/ThemeProvider';
import { useGlasbey } from 'hooks/useGlasbey';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { searcherMetricsValColumn } from 'pages/F_ExpList/glide-table/columns';
import { handlePath } from 'routes/utils';
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import { Project } from 'types';
import handleError from 'utils/error';
import { AnyMouseEvent } from 'utils/routes';

import {
  FlatRunsGlobalSettings,
  FlatRunsSettings,
  settingsConfigForProject,
  settingsConfigGlobal,
} from './FlatRuns.settings';

export const PAGE_SIZE = 100;
const INITIAL_LOADING_RUNS: Loadable<unknown>[] = new Array(PAGE_SIZE).fill(NotLoaded);

const STATIC_COLUMNS = [MULTISELECT];

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
  const { settings: globalSettings, updateSettings: updateGlobalSettings } =
    useSettings<FlatRunsGlobalSettings>(settingsConfigGlobal);

  const [runs, setRuns] = useState<Loadable<unknown>[]>(INITIAL_LOADING_RUNS);
  const isPagedView = globalSettings.tableViewMode === 'paged';
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
  const [total, setTotal] = useState<Loadable<number>>(NotLoaded);

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
  const setSelectAll = useCallback(
    (selectAll: boolean) => {
      updateSettings({ selectAll });
    },
    [updateSettings],
  );

  const selectedRunIds: Set<number> = useMemo(() => {
    return isLoadingSettings ? new Set() : new Set(settings.selectedRuns);
  }, [isLoadingSettings, settings.selectedRuns]);

  const excludedRunIds: Set<number> = useMemo(() => {
    return isLoadingSettings ? new Set() : new Set(settings.excludedRuns);
  }, [isLoadingSettings, settings.excludedRuns]);

  const selectedRuns: unknown[] = useMemo(() => {
    if (selectedRunIds.size === 0) return [];
    return Loadable.filterNotLoaded(runs, (run) => selectedRunIds.has(run.id));
  }, [runs, selectedRunIds]);

  const [isLoading, setIsLoading] = useState(true);
  const [error] = useState(false);
  const [canceler] = useState(new AbortController());

  const colorMap = useGlasbey(settings.selectedRuns);
  //const { height: containerHeight, width: containerWidth } = useResize(contentRef);
  //const height = containerHeight - 2 * parseInt(getThemeVar('strokeWidth')) - (isPagedView ? 40 : 0);
  const [scrollPositionSetCount] = useState(observable(0));

  const {
    ui: { theme: appTheme },
    isDarkMode,
  } = useUI();

  const columns: ColumnDef<unknown>[] = useMemo(() => {
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
          return (columnDefs[columnName] = defaultSelectionColumn(selection.rows, selectAll));
        }
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
          /* case V1LocationType.RUN:
              break;
          case V1LocationType.RUNHYPERPARAMETERS:
            break; */
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
            if (
              heatmap &&
              settings.heatmapOn &&
              !settings.heatmapSkipped.includes(currentColumn.column)
            ) {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn.column,
                currentColumn.displayName || currentColumn.column,
                settings.columnWidths[currentColumn.column],
                dataPath,
                {
                  max: heatmap.max,
                  min: heatmap.min,
                },
              );
            } else {
              columnDefs[currentColumn.column] = defaultNumberColumn(
                currentColumn.column,
                currentColumn.displayName || currentColumn.column,
                settings.columnWidths[currentColumn.column],
                dataPath,
              );
            }
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
        if (currentColumn.column === 'searcherMetricsVal') {
          const heatmap = projectHeatmap.find((h) => h.metricsName === currentColumn.column);
          if (
            heatmap &&
            settings.heatmapOn &&
            !settings.heatmapSkipped.includes(currentColumn.column)
          ) {
            columnDefs[currentColumn.column] = searcherMetricsValColumn(
              settings.columnWidths[currentColumn.column],
              {
                max: heatmap.max,
                min: heatmap.min,
              },
            );
          } else {
            columnDefs[currentColumn.column] = searcherMetricsValColumn(
              settings.columnWidths[currentColumn.column],
            );
          }
        }
        return columnDefs[currentColumn.column];
      })
      .flatMap((col) => (col ? [col] : []));
    return gridColumns;
  }, [
    appTheme,
    columnsIfLoaded,
    isDarkMode,
    selectAll,
    selection.rows,
    settings.columnWidths,
    settings.compare,
    settings.heatmapOn,
    settings.heatmapSkipped,
    settings.pinnedColumnsCount,
  ]);

  const onPageChange = useCallback(
    (cPage: number, cPageSize: number) => {
      updateSettings({ pageLimit: cPageSize });
      // Pagination component is assuming starting index of 1.
      if (cPage - 1 !== page) {
        setRuns(Array(cPageSize).fill(NotLoaded));
      }
      setPage(cPage - 1);
    },
    [page, updateSettings],
  );

  const fetchRuns = useCallback(async (): Promise<void> => {
    if (isLoadingSettings || Loadable.isNotLoaded(loadableFormset)) return;
    try {
      const tableOffset = Math.max((page - 0.5) * PAGE_SIZE, 0);
      // const response = await searchExperiments(
      //   {
      //     ...experimentFilters,
      //     filter: filtersString,
      //     limit: isPagedView ? settings.pageLimit : 2 * PAGE_SIZE,
      //     offset: isPagedView ? page * settings.pageLimit : tableOffset,
      //     sort: sortString || undefined,
      //   },
      //   { signal: canceler.signal },
      // );
      const response = await {
        pagination: {
          total: 0,
        },
        runs: [],
      };
      const total = response.pagination.total ?? 0;
      const loadedRuns = response.runs;

      setRuns((prev) => {
        if (isPagedView) {
          return loadedRuns.map((run) => Loaded(run));
        }

        let newRuns = prev;

        // Fill out the loadable experiments array with total count.
        if (prev.length !== total) {
          newRuns = new Array(total).fill(NotLoaded);
        }

        // Update the list with the fetched results.
        Array.prototype.splice.apply(newRuns, [
          tableOffset,
          loadedRuns.length,
          ...loadedRuns.map((experiment) => Loaded(experiment)),
        ]);

        return newRuns;
      });
      setTotal(
        response.pagination.total !== undefined ? Loaded(response.pagination.total) : NotLoaded,
      );
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [
    canceler.signal,
    experimentFilters,
    filtersString,
    isLoadingSettings,
    isPagedView,
    loadableFormset,
    page,
    sortString,
    settings.pageLimit,
  ]);

  const { stopPolling } = usePolling(fetchRuns, { rerunOnNewFn: true });

  const resetPagination = useCallback(() => {
    setIsLoading(true);
    setPage(0);
    setRuns(INITIAL_LOADING_RUNS);
    setSelection({ columns: CompactSelection.empty(), rows: CompactSelection.empty() });
  }, []);

  const handleTableViewModeChange = useCallback(
    (mode: TableViewMode) => {
      // Reset page index when table view mode changes.
      resetPagination();
      updateGlobalSettings({ tableViewMode: mode });
    },
    [resetPagination, updateGlobalSettings],
  );

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
  }, [page, sortString]);

  const handleColumnWidthChange = useCallback(() => {}, []);

  const handleContextMenuComplete: ContextMenuCompleteHandlerProps<unknown, unknown> = useCallback(
    (action: unknown, id: number, data?: Partial<unknown>) => {},
    [],
  );

  const handleColumnsOrderChange = useCallback(
    (newColumnsOrder: string[]) => {
      updateSettings({ columns: newColumnsOrder });
    },
    [updateSettings],
  );

  return (
    <>
      <DataGrid
        columns={columns}
        data={runs}
        numRows={isPagedView ? runs.length : Loadable.getOrElse(PAGE_SIZE, total)}
        page={page}
        pageSize={PAGE_SIZE}
        scrollPositionSetCount={scrollPositionSetCount}
        selection={selection}
        staticColumns={STATIC_COLUMNS}
        onColumnResize={handleColumnWidthChange}
        onColumnsOrderChange={handleColumnsOrderChange}
        onContextMenuComplete={handleContextMenuComplete}
        onLinkClick={(href) => {
          handlePath(event as unknown as AnyMouseEvent, { path: href });
        }}
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
  );
};

export default FlatRuns;
