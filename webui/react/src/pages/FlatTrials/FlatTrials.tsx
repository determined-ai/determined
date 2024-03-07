import Column from 'hew/Column';
import { MULTISELECT } from 'hew/DataGrid/columns';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
import DataGrid from 'hew/DataGrid/DataGrid';
import Pagination from 'hew/Pagination';
import Row from 'hew/Row';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { handlePath } from 'routes/utils';
import { Project } from 'types';
import handleError from 'utils/error';
import { AnyMouseEvent } from 'utils/routes';

import {
  FlatTrialsGlobalSettings,
  FlatTrialsSettings,
  settingsConfigForProject,
  settingsConfigGlobal,
} from './FlatTrials.settings';

export const PAGE_SIZE = 100;
const INITIAL_LOADING_RUNS: Loadable<unknown>[] = new Array(PAGE_SIZE).fill(NotLoaded);

const STATIC_COLUMNS = [MULTISELECT];

interface Props {
  project: Project;
}

const FlatTrials: React.FC<Props> = ({ project }) => {
  const contentRef = useRef<HTMLDivElement>(null);
  const [searchParams, setSearchParams] = useSearchParams();
  const settingsConfig = useMemo(() => settingsConfigForProject(project.id), [project.id]);
  const {
    isLoading: isLoadingSettings,
    settings,
    updateSettings,
  } = useSettings<FlatTrialsSettings>(settingsConfig);
  const { settings: globalSettings, updateSettings: updateGlobalSettings } =
    useSettings<FlatTrialsGlobalSettings>(settingsConfigGlobal);

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

  const fetchExperiments = useCallback(async (): Promise<void> => {
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
      const response = {
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

  const { stopPolling } = usePolling(fetchExperiments, { rerunOnNewFn: true });

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
        data={runs}
        numRows={isPagedView ? runs.length : Loadable.getOrElse(PAGE_SIZE, total)}
        page={page}
        pageSize={PAGE_SIZE}
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

export default FlatTrials;
