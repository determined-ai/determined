import { Rectangle } from '@glideapps/glide-data-grid';
import { Dropdown, DropDownProps, MenuProps, Row, Space } from 'antd';
import SkeletonButton from 'antd/es/skeleton/Button';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { useSetDynamicTabBar } from 'components/DynamicTabs';
import FilterCounter from 'components/FilterCounter';
import Button from 'components/kit/Button';
import Toggle from 'components/kit/Toggle';
import Page from 'components/Page';
import useModalColumnsCustomize from 'hooks/useModal/Columns/useModalColumnsCustomize';
import useResize from 'hooks/useResize';
import { UpdateSettings, useSettings } from 'hooks/useSettings';
import { getExperiments } from 'services/api';
import { Experimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { GetExperimentsParams } from 'services/types';
import Icon from 'shared/components/Icon/Icon';
import usePolling from 'shared/hooks/usePolling';
import { ValueOf } from 'shared/types';
import { validateDetApiEnum, validateDetApiEnumList } from 'shared/utils/service';
import usersStore from 'stores/users';
import { ExperimentItem, Project, RunState } from 'types';
import handleError from 'utils/error';

import {
  DEFAULT_COLUMN_WIDTHS,
  DEFAULT_COLUMNS,
  ExperimentColumnName,
  ExperimentListSettings,
  settingsConfigForProject,
} from '../ExperimentList.settings';
import css from '../ProjectDetails.module.scss';

import GlideTable from './glide-table/GlideTable';
import { useGlasbey } from './glide-table/useGlasbey';

interface ExplistProps {
  project: Project;
}
const filterKeys: Array<keyof ExperimentListSettings> = ['label', 'search', 'state', 'user'];

const PAGE_RADIUS = 50;
const F_ExperimentList: React.FC<ExplistProps> = ({ project }) => {
  const [pageMidpoint, setPageMidpoint] = useState(0);

  const handleScroll = useCallback(({ y, height }: Rectangle) => {
    const visibleRegionMidPoint = y + height / 2;
    setPageMidpoint(Math.round(visibleRegionMidPoint / PAGE_RADIUS) * PAGE_RADIUS);
  }, []);

  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);

  const [isLoading, setIsLoading] = useState(true);

  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const { width } = useResize(pageRef);

  const id = project?.id;

  const settingsConfig = useMemo(() => settingsConfigForProject(id), [id]);

  const { settings, updateSettings, resetSettings, activeSettings } =
    useSettings<ExperimentListSettings>(settingsConfig);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [activeSettings]);

  const statesString = useMemo(() => settings.state?.join('.'), [settings.state]);
  const pinnedString = useMemo(() => JSON.stringify(settings.pinned ?? {}), [settings.pinned]);
  const labelsString = useMemo(() => settings.label?.join('.'), [settings.label]);
  const usersString = useMemo(() => settings.user?.join('.'), [settings.user]);

  const fetchExperiments = useCallback(
    async (): Promise<void> => {
      if (!settings) return;
      try {
        const states = statesString
          ?.split('.')
          .map((state) => encodeExperimentState(state as RunState));
        // const pinned = JSON.parse(pinnedString);
        const baseParams: GetExperimentsParams = {
          archived: settings.archived ? undefined : false,
          labels: settings.label,
          name: settings.search,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          projectId: id,
          sortBy: validateDetApiEnum(V1GetExperimentsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(Experimentv1State, states),
          users: settings.user,
        };

        const tableLimit = 2 * PAGE_RADIUS;
        const tableOffset = Math.max(pageMidpoint - PAGE_RADIUS, 0);

        const expResponse = await getExperiments(
          {
            ...baseParams,
            limit: tableLimit,
            offset: tableOffset,
          },
          { signal: canceler.signal },
        );

        setExperiments((prevExperiments) => {
          const newExperiments = [];
          let i = 0;
          while (i < tableOffset) {
            // TODO: coalesce to emptyExperiment to prevent possible crash
            newExperiments[i] = prevExperiments[i];
            i++;
          }
          while (expResponse.experiments[i - tableOffset]) {
            newExperiments[i] = expResponse.experiments[i - tableOffset];
            i++;
          }
          while (prevExperiments[i]) {
            newExperiments[i] = prevExperiments[i];
            i++;
          }
          return newExperiments;
        });
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch experiments.' });
      } finally {
        setIsLoading(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      canceler.signal,
      id,
      settings,
      labelsString,
      pinnedString,
      statesString,
      usersString,
      pageMidpoint,
    ],
  );

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchExperiments(), usersStore.ensureUsersFetched(canceler)]);
  }, [fetchExperiments, canceler]);

  const { stopPolling } = usePolling(fetchAll, { rerunOnNewFn: true });

  const columns = useMemo(
    () => [
      'id',
      'name',
      'description',
      'tags',
      'forkedFrom',
      'startTime',
      'duration',
      'numTrials',
      'state',
      'searcherType',
      'resourcePool',
      'progress',
      'checkpointSize',
      'checkpointCount',
      'archived',
      'user',
      'action',
      'searcherMetricValue',
    ],
    [],
  );

  const transferColumns = useMemo(() => {
    return columns.filter((column) => !['action', 'archived'].includes(column));
  }, [columns]);

  const initialVisibleColumns = useMemo(
    () => settings.columns?.filter((col) => transferColumns.includes(col)),
    [settings.columns, transferColumns],
  );

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [updateSettings]);

  const resetFilters = useCallback(() => {
    resetSettings([...filterKeys, 'tableOffset']);
    clearSelected();
  }, [clearSelected, resetSettings]);

  const handleUpdateColumns = useCallback(
    (columns: ExperimentColumnName[]) => {
      if (columns.length === 0) {
        updateSettings({
          columns: ['id', 'name'],
          columnWidths: [DEFAULT_COLUMN_WIDTHS['id'], DEFAULT_COLUMN_WIDTHS['name']],
        });
      } else {
        updateSettings({
          columns: columns,
          columnWidths: columns.map((col) => DEFAULT_COLUMN_WIDTHS[col]),
        });
      }
    },
    [updateSettings],
  );

  const { contextHolder: modalColumnsCustomizeContextHolder, modalOpen: openCustomizeColumns } =
    useModalColumnsCustomize({
      columns: transferColumns,
      defaultVisibleColumns: DEFAULT_COLUMNS,
      initialVisibleColumns,
      onSave: handleUpdateColumns as (columns: string[]) => void,
    });

  const handleCustomizeColumnsClick = useCallback(() => {
    openCustomizeColumns({});
  }, [openCustomizeColumns]);

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      if (!settings) return;
      let newColumns: ExperimentColumnName[];
      let newColumnWidths: number[];

      if (showArchived) {
        if (settings.columns?.includes('archived')) {
          // just some defensive coding: don't add archived twice
          newColumns = settings.columns;
          newColumnWidths = settings.columnWidths;
        } else {
          newColumns = [...settings.columns, 'archived'];
          newColumnWidths = [...settings.columnWidths, DEFAULT_COLUMN_WIDTHS['archived']];
        }
      } else {
        const archivedIndex = settings.columns.indexOf('archived');
        if (archivedIndex !== -1) {
          newColumns = [...settings.columns];
          newColumnWidths = [...settings.columnWidths];
          newColumns.splice(archivedIndex, 1);
          newColumnWidths.splice(archivedIndex, 1);
        } else {
          newColumns = settings.columns;
          newColumnWidths = settings.columnWidths;
        }
      }
      updateSettings({
        archived: showArchived,
        columns: newColumns,
        columnWidths: newColumnWidths,
        row: undefined,
      });
    },
    [settings, updateSettings],
  );

  useEffect(() => {
    setIsLoading(true);
    fetchExperiments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  const tabBarContent = useMemo(() => {
    const getMenuProps = (): DropDownProps['menu'] => {
      const MenuKey = {
        Columns: 'columns',
        ResultFilter: 'resetFilters',
        SwitchArchived: 'switchArchive',
      } as const;

      const funcs = {
        [MenuKey.SwitchArchived]: () => {
          switchShowArchived(!settings.archived);
        },
        [MenuKey.Columns]: () => {
          handleCustomizeColumnsClick();
        },
        [MenuKey.ResultFilter]: () => {
          resetFilters();
        },
      };

      const onItemClick: MenuProps['onClick'] = (e) => {
        funcs[e.key as ValueOf<typeof MenuKey>]();
      };

      const menuItems: MenuProps['items'] = [
        {
          key: MenuKey.SwitchArchived,
          label: settings.archived ? 'Hide Archived' : 'Show Archived',
        },
        { key: MenuKey.Columns, label: 'Columns' },
      ];
      if (filterCount > 0) {
        menuItems.push({ key: MenuKey.ResultFilter, label: `Clear Filters (${filterCount})` });
      }
      return { items: menuItems, onClick: onItemClick };
    };
    return (
      <div className={css.tabOptions}>
        <Space className={css.actionList}>
          <Toggle checked={settings.archived} label="Show Archived" onChange={switchShowArchived} />
          <Button onClick={handleCustomizeColumnsClick}>Columns</Button>
          <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
        </Space>
        <div className={css.actionOverflow} title="Open actions menu">
          <Dropdown menu={getMenuProps()} trigger={['click']}>
            <div>
              <Icon name="overflow-vertical" />
            </div>
          </Dropdown>
        </div>
      </div>
    );
  }, [
    filterCount,
    handleCustomizeColumnsClick,
    resetFilters,
    settings.archived,
    switchShowArchived,
  ]);

  const [columnIds, setColumnIds] = useState(settings.columns as string[]);
  useEffect(() => {
    // updateSettings({ columns });
  }, [columns, updateSettings]);

  useSetDynamicTabBar(tabBarContent);

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const colorMap = useGlasbey(selectedIds);

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <>
        {isLoading ? (
          [...Array(22)].map((x, i) => (
            <Row key={i} style={{ paddingBottom: '4px' }}>
              <SkeletonButton style={{ width: width - 20 }} />
            </Row>
          ))
        ) : (
          <GlideTable
            colorMap={colorMap}
            columns={columnIds}
            data={experiments}
            handleScroll={handleScroll}
            selectedIds={selectedIds}
            setColumns={setColumnIds}
            setSelectedIds={setSelectedIds}
            settings={settings}
            updateSettings={updateSettings as UpdateSettings}
          />
        )}
      </>
      {modalColumnsCustomizeContextHolder}
    </Page>
  );
};

export default F_ExperimentList;
