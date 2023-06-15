import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { MutableRefObject, useCallback, useEffect, useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import BadgeTag from 'components/BadgeTag';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  getFullPaginationConfig,
  relativeTimeRenderer,
  userRenderer,
} from 'components/Table/Table';
import TableFilterMultiSearch from 'components/Table/TableFilterMultiSearch';
import TableFilterRank from 'components/Table/TableFilterRank';
import { Highlights } from 'hooks/useHighlight';
import { UseSettingsReturn } from 'hooks/useSettings';
import { TrialsWithMetadata } from 'pages/TrialsComparison/Trials/data';
import { paths } from 'routes/utils';
import { Trialv1State, V1AugmentedTrial } from 'services/api-ts-sdk';
import userStore from 'stores/users';
import { StateOfUnion } from 'themes';
import { MetricType } from 'types';
import { ColorScale, glasbeyColor } from 'utils/color';
import { isFiniteNumber } from 'utils/data';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { getDisplayName } from 'utils/user';

import { TrialActionsInterface } from '../Actions/useTrialActions';
import { TrialsCollectionInterface } from '../Collections/useTrialCollections';

import rangeFilterForPrefix, { rangeFilterIsActive } from './rangeFilter';
import trialTagsRenderer from './tagsRenderer';
import css from './TrialTable.module.scss';

const NUMBER_PRECISION = 3;
interface Props {
  actionsInterface: TrialActionsInterface;
  collectionsInterface: TrialsCollectionInterface;
  colorScale?: ColorScale[];
  containerRef: MutableRefObject<HTMLElement | null>;
  highlights: Highlights<V1AugmentedTrial>;
  loading?: boolean;
  tableSettingsHook: UseSettingsReturn<InteractiveTableSettings>;
  trialsWithMetadata: TrialsWithMetadata;
}

const getWidthForField = (field: string) => field.length * 8 + 80;
const nonRankableColumns = [
  'experimentId',
  'experimentName',
  'userId',
  'rank',
  'searcherMetric',
  'searcherMetricLoss',
  'searcherType',
  'tags',
  'state',
];

const TrialTable: React.FC<Props> = ({
  // colorScale,
  collectionsInterface,
  actionsInterface,
  trialsWithMetadata: trials,
  containerRef,
  highlights,
  tableSettingsHook,
}: Props) => {
  const { settings, updateSettings } = tableSettingsHook;

  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));

  const { filters, setFilters } = collectionsInterface;

  const { TrialActionDropdown, selectAllMatching, selectedTrials, selectTrials } = actionsInterface;

  const idColumn = useMemo(
    () => ({
      defaultWidth: 100,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterMultiSearch
          {...filterProps}
          multiple
          searchable
          validatorRegex={/\D/}
          values={filters.trialIds}
          onFilter={(trialIds: string[]) => setFilters?.((filters) => ({ ...filters, trialIds }))}
          onReset={() => setFilters?.((filters) => ({ ...filters, trialIds: [] }))}
        />
      ),
      isFiltered: () => !!filters.trialIds?.length,
      key: 'trialId',
      render: (_: string, record: V1AugmentedTrial) => {
        const color = glasbeyColor(record.trialId);
        return (
          <div className={css.idLayout}>
            <div className={css.colorLegend} style={{ backgroundColor: color }} />
            <Link path={paths.trialDetails(record.trialId, record.experimentId)}>
              {record.trialId}
            </Link>
          </div>
        );
      },
      sorter: true,
      title: 'Trial ID',
    }),
    [filters.trialIds, setFilters],
  );

  const experimentIdColumn = useMemo(
    () => ({
      defaultWidth: 130,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterMultiSearch
          {...filterProps}
          multiple
          searchable
          validatorRegex={/\D/}
          values={filters.experimentIds}
          onFilter={(experimentIds: string[]) =>
            setFilters?.((filters) => ({ ...filters, experimentIds }))
          }
          onReset={() => setFilters?.((filters) => ({ ...filters, experimentIds: [] }))}
        />
      ),
      isFiltered: () => !!filters.experimentIds?.length,
      key: 'experimentId',
      render: (_: string, record: V1AugmentedTrial) => (
        <Link path={paths.experimentDetails(record.experimentId)}>{record.experimentId}</Link>
      ),
      sorter: true,
      title: 'Exp ID',
    }),
    [filters.experimentIds, setFilters],
  );

  const expRankColumn = useMemo(
    () => ({
      defaultWidth: 160,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterRank
          {...filterProps}
          column={filters.ranker?.sorter.sortKey}
          columns={settings.columns.filter((col) => !nonRankableColumns.includes(col))}
          rank={filters.ranker?.rank}
          sortDesc={filters.ranker?.sorter.sortDesc ?? false}
          onReset={() =>
            setFilters?.((filters) => ({
              ...filters,
              ranker: {
                rank: '',
                sorter: { sortDesc: false, sortKey: 'searcherMetricValue' },
              },
            }))
          }
          onSet={(column, sortDesc, rank) => {
            setFilters?.((filters) => ({
              ...filters,
              ranker: { rank, sorter: { sortDesc, sortKey: column } },
            }));
          }}
        />
      ),
      isFiltered: () => (filters.ranker?.rank ? parseInt(filters.ranker?.rank) !== 0 : false),
      key: 'rank',
      render: (_: string, record: V1AugmentedTrial) => (
        <div className={css.idLayout}>{record.rankWithinExp}</div>
      ),
      sorter: true,
      title: 'Rank in Exp',
    }),
    [
      filters.ranker?.rank,
      setFilters,
      settings.columns,
      filters.ranker?.sorter.sortDesc,
      filters.ranker?.sorter.sortKey,
    ],
  );

  const hpColumns = useMemo(
    () =>
      Object.keys(trials.hparams ?? {})
        // .filter((hp) => trials.hparams[hp]?.size > 1)
        .map((hp) => {
          const columnKey = `hparams.${hp}`;
          const actionable = [...trials.hparams[hp]].some((val) =>
            isFiniteNumber(parseFloat(String(val))),
          );
          return {
            defaultWidth: getWidthForField(hp),
            filterDropdown: actionable
              ? rangeFilterForPrefix('hparams', filters, setFilters)(hp)
              : null,
            isFiltered: () => rangeFilterIsActive(filters.hparams, hp),
            key: columnKey,
            render: (_: string, record: V1AugmentedTrial) => {
              const value = record.hparams[hp];
              if (isFiniteNumber(value)) {
                return <HumanReadableNumber num={value} precision={3} />;
              } else if (!value) {
                return '-';
              }
              return value + '';
            },
            sorter: actionable,
            title: (
              <BadgeTag label={hp} tooltip="Hyperparameter">
                H
              </BadgeTag>
            ),
          };
        }),
    [filters, trials.hparams, setFilters],
  );

  const tagColumn = useMemo(
    () => ({
      defaultWidth: 200,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterMultiSearch
          {...filterProps}
          multiple
          placeholder="Filter Tags"
          searchable
          validatorRegex={/[^a-zA-Z0-9]+$/} // TODO need fix ?
          values={filters.tags}
          onFilter={(tags) => setFilters?.((filters) => ({ ...filters, tags }))}
          onReset={() => setFilters?.((filters) => ({ ...filters, tags: [] }))}
        />
      ),
      isFiltered: () => !!filters.tags?.length,
      key: 'tags',
      render: trialTagsRenderer,
      sorter: false,
      title: 'Tags',
    }),
    [filters.tags, setFilters],
  );

  const trainingMetricColumns = useMemo(
    () =>
      trials.metrics
        .filter((metric) => metric.type === MetricType.Training)
        .map((metric) => {
          const columnKey = `trainingMetrics.${metric.name}`;
          return {
            defaultWidth: getWidthForField(metric.name),
            filterDropdown: rangeFilterForPrefix(
              'trainingMetrics',
              filters,
              setFilters,
            )(metric.name),
            isFiltered: () => rangeFilterIsActive(filters.trainingMetrics, metric.name),
            key: columnKey,
            render: (_: string, record: V1AugmentedTrial) => {
              const value = record.trainingMetrics?.[metric.name];
              return isFiniteNumber(value) ? (
                <HumanReadableNumber num={value} precision={NUMBER_PRECISION} />
              ) : (
                value ?? '-'
              );
            },
            sorter: true,
            title: <MetricBadgeTag metric={metric} />,
          };
        }),
    [filters, trials.metrics, setFilters],
  );

  const validationMetricColumns = useMemo(
    () =>
      trials.metrics
        .filter((metric) => metric.type === MetricType.Validation)
        .map((metric) => {
          const columnKey = `validationMetrics.${metric.name}`;
          return {
            defaultWidth: getWidthForField(metric.name),
            filterDropdown: rangeFilterForPrefix(
              'validationMetrics',
              filters,
              setFilters,
            )(metric.name),
            isFiltered: () => rangeFilterIsActive(filters.validationMetrics, metric.name),
            key: columnKey,
            render: (_: string, record: V1AugmentedTrial) => {
              const value = record.validationMetrics?.[metric.name];
              return isFiniteNumber(value) ? (
                <HumanReadableNumber num={value} precision={NUMBER_PRECISION} />
              ) : (
                value ?? '-'
              );
            },
            sorter: true,
            title: <MetricBadgeTag metric={metric} />,
          };
        }),
    [filters, trials.metrics, setFilters],
  );

  const stateColumn = useMemo(
    () => ({
      defaultWidth: 100,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterMultiSearch
          {...filterProps}
          multiple
          values={filters.states ?? []}
          onFilter={(states) => setFilters?.((filters) => ({ ...filters, states }))}
          onReset={() => setFilters?.((filters) => ({ ...filters, states: undefined }))}
        />
      ),
      filters: [
        Trialv1State.ACTIVE,
        Trialv1State.PAUSED,
        Trialv1State.CANCELED,
        Trialv1State.COMPLETED,
        Trialv1State.ERROR,
      ].map((value) => ({
        text: <Badge state={value} type={BadgeType.State} />,
        value,
      })),
      isFiltered: () => !!filters?.states?.length,
      key: 'state',
      render: (_: string, record: V1AugmentedTrial) => (
        <div className={css.centerVertically}>
          <Badge state={record.state as unknown as StateOfUnion} type={BadgeType.State} />
        </div>
      ),
      sorter: true,
      title: 'State',
    }),
    [setFilters, filters],
  );

  const startTimeColumn = useMemo(
    () => ({
      defaultWidth: 120,
      key: 'startTime',
      render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
        relativeTimeRenderer(new Date(record.startTime)),
      sorter: true,
      title: 'Started',
    }),
    [],
  );

  const endTimeColumn = useMemo(
    () => ({
      defaultWidth: 100,
      key: 'endTime',
      render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
        relativeTimeRenderer(new Date(record.startTime)),
      sorter: true,
      title: 'Ended',
    }),
    [],
  );

  const experimentNameColumn = useMemo(
    () => ({
      defaultWidth: 160,
      key: 'experimentName',
      render: (value: string | number | undefined, record: V1AugmentedTrial): React.ReactNode => (
        <Link path={paths.experimentDetails(record.experimentId)}>
          {value === undefined ? '' : value}
        </Link>
      ),
      title: 'Experiment Name',
    }),
    [],
  );

  const searcherTypeColumn = useMemo(
    () => ({
      defaultWidth: 120,
      key: 'searcherType',
      title: (
        <BadgeTag label="Type" tooltip="Searcher">
          S
        </BadgeTag>
      ),
    }),
    [],
  );

  const searcherMetricColumn = useMemo(
    () => ({
      defaultWidth: 125,
      key: 'searcherMetric',
      sorter: true,
      title: (
        <BadgeTag label="Metric" tooltip="Searcher Metric">
          S
        </BadgeTag>
      ),
    }),
    [],
  );

  const userColumn = useMemo(() => {
    return {
      defaultWidth: 100,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterMultiSearch
          {...filterProps}
          multiple
          searchable
          values={filters.userIds}
          onFilter={(userIds: string[]) => setFilters?.((filters) => ({ ...filters, userIds }))}
          onReset={() => setFilters?.((filters) => ({ ...filters, userIds: undefined }))}
        />
      ),
      filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
      isFiltered: () => !!filters.userIds?.length,
      key: 'userId',
      render: (_: number, r: V1AugmentedTrial) =>
        userRenderer(users.find((u) => u.id === r.userId)),
      sorter: true,
      title: 'User',
    };
  }, [filters.userIds, setFilters, users]);

  const totalBatchesColumn = useMemo(
    () => ({
      defaultWidth: 130,
      key: 'totalBatches',
      sorter: true,
      title: 'Total Batches',
    }),
    [],
  );

  const searcherMetricValueColumn = useMemo(
    () => ({
      defaultWidth: 110,
      key: 'searcherMetricValue',
      render: (_: string, record: V1AugmentedTrial) => (
        <HumanReadableNumber num={record.searcherMetricValue} precision={NUMBER_PRECISION} />
      ),
      sorter: true,
      title: (
        <BadgeTag label="Value" tooltip="Searcher Metric Value">
          S
        </BadgeTag>
      ),
    }),
    [],
  );

  const ContextMenu = useCallback(
    ({
      record,
      children,
    }: {
      children?: React.ReactNode;
      onVisibleChange?: ((visible: boolean) => void) | undefined;
      record: V1AugmentedTrial;
    }) => {
      return <TrialActionDropdown id={record.trialId}>{children}</TrialActionDropdown>;
    },
    [TrialActionDropdown],
  );

  const actionColumn = useMemo(
    () => ({
      align: 'right',
      className: 'fullCell',
      defaultWidth: 110,
      fixed: 'right',
      key: 'action',
      render: (_: string, record: V1AugmentedTrial) => <TrialActionDropdown id={record.trialId} />,

      title: '',
      width: 80,
    }),
    [TrialActionDropdown],
  );

  const columns = useMemo(() => {
    return [
      experimentNameColumn,
      experimentIdColumn,
      idColumn,
      expRankColumn,
      searcherTypeColumn,
      searcherMetricColumn,
      searcherMetricValueColumn,
      ...validationMetricColumns,
      ...trainingMetricColumns,
      tagColumn,
      userColumn,
      totalBatchesColumn,
      stateColumn,
      startTimeColumn,
      endTimeColumn,
      ...hpColumns,
      actionColumn,
    ].map((col) => ({ ...col, dataIndex: col.key } as ColumnDef<V1AugmentedTrial>));
  }, [
    actionColumn,
    idColumn,
    experimentIdColumn,
    expRankColumn,
    tagColumn,
    hpColumns,
    trainingMetricColumns,
    validationMetricColumns,
    userColumn,
    totalBatchesColumn,
    searcherMetricValueColumn,
    searcherMetricColumn,
    searcherTypeColumn,
    stateColumn,
    experimentNameColumn,
    startTimeColumn,
    endTimeColumn,
  ]);

  const availableColumns = useMemo(
    () => (columns.map((c) => String(c.key)).slice(0, -1) ?? []).join('|'),
    [columns],
  );

  useEffect(() => {
    updateSettings({
      columns: columns.map((c) => String(c.key)).slice(0, -1) ?? [],
      columnWidths: columns.map((c) => c.defaultWidth),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [availableColumns]);

  const pagination = useMemo(() => {
    const limit = settings.tableLimit || 0;
    const offset = settings.tableOffset || 0;
    // we fetched 3 * params.limit to be able to see ahead like this
    // also has bonus side effect of giving us a better hit rate
    // for the relevant dynamic columns
    const fakeTotal = offset + trials.data.length;
    return getFullPaginationConfig(
      {
        limit,
        offset,
      },
      fakeTotal,
    );
  }, [settings.tableLimit, settings.tableOffset, trials.data.length]);

  return (
    <div className={css.base}>
      {settings ? (
        <InteractiveTable<V1AugmentedTrial>
          columns={columns}
          containerRef={containerRef}
          ContextMenu={ContextMenu}
          dataSource={trials.data.slice(0, settings.tableLimit)}
          interactiveColumns={false}
          loading={userColumn === undefined}
          pagination={pagination}
          rowClassName={highlights.rowClassName}
          rowKey="trialId"
          rowSelection={{
            getCheckboxProps: () => ({ disabled: selectAllMatching }),
            onChange: selectTrials,
            preserveSelectedRowKeys: true,
            selectedRowKeys: (selectAllMatching ? trials.ids : selectedTrials) as number[],
          }}
          scroll={{ x: 'max-content', y: '40vh' }}
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
          onRow={highlights.onRow}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
    </div>
  );
};

export default TrialTable;
