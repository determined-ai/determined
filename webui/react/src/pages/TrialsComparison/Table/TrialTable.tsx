import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { MutableRefObject, useEffect, useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import BadgeTag from 'components/BadgeTag';
import HumanReadableNumber from 'components/HumanReadableNumber';
import InteractiveTable, { InteractiveTableSettings } from 'components/InteractiveTable';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import { getFullPaginationConfig, getPaginationConfig, relativeTimeRenderer } from 'components/Table';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
import UserAvatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import { Highlights } from 'hooks/useHighlight';
import { SettingsHook, UpdateSettings } from 'hooks/useSettings';
import { TrialsWithMetadata } from 'pages/TrialsComparison/Trials/data';
import { paths } from 'routes/utils';
import { Determinedtrialv1State, V1AugmentedTrial } from 'services/api-ts-sdk';
import { ColorScale, glasbeyColor } from 'shared/utils/color';
import { isNumber } from 'shared/utils/data';
import { StateOfUnion } from 'themes';
import { MetricType, RunState, TrialState } from 'types';
import { metricKeyToName, metricToKey } from 'utils/metric';
import { getDisplayName } from 'utils/user';

import { TrialActionsInterface } from '../Actions/useTrialActions';
import { TrialSorter } from '../Collections/filters';
import { TrialsCollectionInterface } from '../Collections/useTrialCollections';

import rangeFilterForPrefix, { rangeFilterIsActive } from './rangeFilter';
import trialTagsRenderer from './tagsRenderer';
import css from './TrialTable.module.scss';

interface Props {
  actionsInterface: TrialActionsInterface
  collectionsInterface: TrialsCollectionInterface;
  colorScale?: ColorScale[];
  containerRef : MutableRefObject<HTMLElement | null>,
  highlights: Highlights<V1AugmentedTrial>;
  tableSettingsHook: SettingsHook<InteractiveTableSettings>;
  trialsWithMetadata: TrialsWithMetadata;
}

const TrialTable: React.FC<Props> = ({
  // colorScale,
  collectionsInterface: C,
  actionsInterface: A,
  trialsWithMetadata: trials,
  containerRef,
  highlights,
  tableSettingsHook,
}: Props) => {

  const { settings, updateSettings } = tableSettingsHook;

  const { users } = useStore();

  const idColumn = useMemo(() => ({
    dataIndex: 'id',
    defaultWidth: 60,
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
  }), []);

  const experimentIdColumn = useMemo(
    () => ({
      dataIndex: 'experimentId',
      defaultWidth: 130,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterDropdown
          {...filterProps}
          multiple
          searchable
          validatorRegex={/\D/}
          values={C.filters.experimentIds}
          onFilter={(experimentIds: string[]) =>
            C.setFilters?.((filters) => ({ ...filters, experimentIds }))
          }

          onReset={() => C.setFilters?.((filters) => ({ ...filters, experimentIds: [] }))}
        />
      ),
      isFiltered: () => !!C.filters.experimentIds?.length,
      key: 'experimentId',
      render: (_: string, record: V1AugmentedTrial) => (
        <Link path={paths.experimentDetails(record.experimentId)}>{record.experimentId}</Link>
      ),
      sorter: true,
      title: 'Exp ID',
    }),
    [ C.filters.experimentIds, C.setFilters ],
  );

  const trialStateColumn = useMemo(
    () => ({
      dataIndex: 'state',
      defaultWidth: 130,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterDropdown
          {...filterProps}
          multiple
          values={C.filters.state}
          onFilter={(state: string[]) =>
            C.setFilters?.((filters) => ({ ...filters, state }))
          }
          onReset={() => C.setFilters?.((filters) => ({ ...filters, state: [] }))}
        />
      ),
      filters: [
        TrialState.ACTIVEUNSPECIFIED,
        TrialState.PAUSED,
        TrialState.CANCELED,
        TrialState.COMPLETED,
        TrialState.ERROR,
      ].map((value) => ({
        text: <Badge state={value} type={BadgeType.State} />,
        value,
      })),
      isFiltered: () => !!C.filters.state?.length,
      key: 'state',
      render: (_: string, record: V1AugmentedTrial) => {
        const apiStateToTrialStateMap: Record< Determinedtrialv1State, TrialState> = {
          [Determinedtrialv1State.ACTIVEUNSPECIFIED]: TrialState.ACTIVEUNSPECIFIED,
          [Determinedtrialv1State.PAUSED]: TrialState.PAUSED,
          [Determinedtrialv1State.STOPPINGCANCELED]: TrialState.STOPPINGCANCELED,
          [Determinedtrialv1State.STOPPINGKILLED]: TrialState.STOPPINGKILLED,
          [Determinedtrialv1State.STOPPINGCOMPLETED]: TrialState.STOPPINGCOMPLETED,
          [Determinedtrialv1State.STOPPINGERROR]: TrialState.STOPPINGERROR,
          [Determinedtrialv1State.CANCELED]: TrialState.CANCELED,
          [Determinedtrialv1State.COMPLETED]: TrialState.COMPLETED,
          [Determinedtrialv1State.ERROR]: TrialState.ERROR,
        };
        return <Badge state={apiStateToTrialStateMap[record.state]} type={BadgeType.State} />;
      },
      sorter: true,
      title: 'State',
    }),
    [ C.filters.state, C.setFilters ],
  );

  const expRankColumn = useMemo(
    () => ({
      dataIndex: 'rank',
      defaultWidth: 60,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterSearch
          {...filterProps}
          value={C.filters.ranker?.rank || ''}
          onReset={() =>
            C.setFilters?.((filters) => ({
              ...filters,
              // TODO handle invalid type assertion below
              ranker: { rank: '', sorter: filters.ranker?.sorter as TrialSorter },
            }))
          }
          onSearch={(r) =>
            C.setFilters?.((filters) => ({
              ...filters,
              // TODO handle invalid type assertion below
              ranker: { rank: r, sorter: filters.ranker?.sorter as TrialSorter },
            }))
          }
        />
      ),
      isFiltered: () => !!C.filters.ranker?.rank,
      key: 'rank',
      render: (_: string, record: V1AugmentedTrial) => (
        <div className={css.idLayout}>{record.rankWithinExp}</div>
      ),
      sorter: true,
      title: 'Rank in Exp',
    }),
    [ C.filters.ranker?.rank, C.setFilters ],
  );

  const hpColumns = useMemo(() => Object
    .keys(trials.hparams || {})
    // .filter((hp) => trials.hparams[hp]?.size > 1)
    .map((hp) => {
      return {
        dataIndex: hp,
        defaultWidth: 130,
        filterDropdown: rangeFilterForPrefix('hparams', C.filters, C.setFilters)(hp),
        isFiltered: () => rangeFilterIsActive(C.filters, 'hparams', hp),
        key: `hparams.${hp}`,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.hparams[hp];
          if (isNumber(value)) {
            return <HumanReadableNumber num={value} />;
          } else if (!value) {
            return '-';
          }
          return value + '';
        },
        sorter: true,
        title: <BadgeTag label={hp} tooltip="Hyperparameter">H</BadgeTag>,
      };
    }), [ C.filters, trials.hparams, C.setFilters ]);

  const tagColumn = useMemo(() => ({
    dataIndex: 'tags',
    defaultWidth: 60,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        placeholder="Filter Tags"
        searchable
        validatorRegex={/[^a-zA-Z0-9]+$/} // TODO need fix ?
        values={C.filters.tags}
        onReset={() => C.setFilters?.((filters) => ({ ...filters, tags: [] }))}
      />
    ),
    isFiltered: () => !!C.filters.tags?.length,
    key: 'labels',
    render: trialTagsRenderer,
    sorter: false,
    title: 'Tags',
  }), [ C.filters.tags, C.setFilters ]);

  const trainingMetricColumns = useMemo(() => trials.metrics
    .filter((metric) => metric.type = MetricType.Training).map((metric) => {
      const key = metricToKey(metric);
      return {
        dataIndex: key,
        defaultWidth: 100,
        filterDropdown: rangeFilterForPrefix(
          'trainingMetrics',
          C.filters,
          C.setFilters,
        )(metric.name),
        isFiltered: () => rangeFilterIsActive(C.filters, 'trainingMetrics', metric.name),
        key: `trainingMetrics.${metric.name}`,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.trainingMetrics?.[metricKeyToName(key)];
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,

      };
    }), [ C.filters, trials.metrics, C.setFilters ]);

  const validationMetricColumns = useMemo(() => trials.metrics
    .filter((metric) => metric.type = MetricType.Validation).map((metric) => {
      const key = metricToKey(metric);
      return {
        dataIndex: key,
        defaultWidth: 100,
        filterDropdown: rangeFilterForPrefix(
          'validationMetrics',
          C.filters,
          C.setFilters,
        )(metric.name),
        isFiltered: () => rangeFilterIsActive(C.filters, 'validationMetrics', metric.name),
        key: `validationMetrics.${metric.name}`,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.validationMetrics?.[metricKeyToName(key)];
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,
      };
    }), [ C.filters, trials.metrics, C.setFilters ]);

  const stateColumn = useMemo(() => ({
    dataIndex: 'state',
    defaultWidth: 80,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={C.filters.states ?? []}
        onFilter={(states) => C.setFilters?.((filters) => ({ ...filters, states }))}
        onReset={() => C.setFilters?.((filters) => ({ ...filters, states: undefined }))}
      />
    ),
    filters: Object.values(RunState)
      .filter((value) => [
        RunState.Active,
        RunState.Paused,
        RunState.Canceled,
        RunState.Completed,
        RunState.Errored,
      ].includes(value))
      .map((value) => ({
        text: <Badge state={value} type={BadgeType.State} />,
        value,
      })),
    isFiltered: () => !!C.filters?.states?.length,
    key: 'states',
    render: (_: string, record: V1AugmentedTrial) => (
      <div className={css.centerVertically}>
        <Badge state={record.state as unknown as StateOfUnion} type={BadgeType.State} />
      </div>
    ),
    sorter: true,
    title: 'State',
  }), []);

  const startTimeColumn = useMemo(() => ({
    dataIndex: 'startTime',
    defaultWidth: 80,
    key: 'startTime',
    render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  }), []);

  const endTimeColumn = useMemo(() => ({
    dataIndex: 'startTime',
    defaultWidth: 80,
    key: 'startTime',
    render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  }), []);

  const experimentNameColumn = useMemo(() => ({
    dataIndex: 'experimentName',
    defaultWidth: 80,
    key: 'experimentName',
    render: (value: string | number | undefined, record: V1AugmentedTrial): React.ReactNode => (
      <Link path={paths.experimentDetails(record.experimentId)}>
        {value === undefined ? '' : value}
      </Link>
    ),
    sorter: true,
    title: 'Name',
  }), []);

  const searcherTypeColumn = useMemo(() => ({
    dataIndex: 'searcherType',
    defaultWidth: 80,
    key: 'searcherType',
    title: 'Searcher Type',
  }), []);

  const searcherMetricColumn = useMemo(() => ({
    dataIndex: 'searcherMetric',
    key: 'searcherMetric',
    sorter: true,
    title: (
      <BadgeTag
        label="Searcher Metric"
        tooltip="Searcher Metric">
        S
      </BadgeTag>
    ),
  }), []);

  const userColumn = useMemo(() => ({
    dataIndex: 'user',
    defaultWidth: 80,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        values={C.filters.userIds}
        onFilter={(userIds: string[]) => C.setFilters?.((filters) => ({ ...filters, userIds }))}
        onReset={() => C.setFilters?.((filters) => ({ ...filters, userIds: undefined }))}
      />
    ),
    filters: users.map((user) => ({ text: getDisplayName(user), value: user.username })),
    isFiltered: () => !!C.filters.userIds,
    key: 'userId',
    render: (_: string, record: V1AugmentedTrial) => <UserAvatar userId={record.userId} />,
    sorter: true,
    title: 'User',
  }), []);

  const totalBatchesColumn = useMemo(() => ({
    dataIndex: 'totalBatches',
    key: 'totalBatches',
    sorter: true,
    title: 'Total Batches',
  }), []);

  const searcherMetricValueColumn = useMemo(() => ({
    dataIndex: 'searcherMetricValue',
    key: 'searcherMetricValue',
    render: (_: string, record: V1AugmentedTrial) => (
      <HumanReadableNumber num={record.searcherMetricValue} />
    ),
    sorter: true,
    title: (
      <BadgeTag
        label="Searcher Metric Value"
        tooltip="Searcher Metric">
        S
      </BadgeTag>
    ),
  }), []);

  // console.log(metrics);

  const columns = useMemo(() => [
    idColumn,
    experimentIdColumn,
    expRankColumn,
    tagColumn,
    userColumn,
    totalBatchesColumn,
    searcherMetricValueColumn,
    searcherMetricColumn,
    searcherTypeColumn,
    experimentNameColumn,
    stateColumn,
    startTimeColumn,
    endTimeColumn,
    ...hpColumns,
    ...trainingMetricColumns,
    ...validationMetricColumns,
  ].map((col) => ({ defaultWidth: 80, ...col })), [
    idColumn,
    experimentIdColumn,
    expRankColumn,
    tagColumn,
    trialStateColumn,
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

  useEffect(() => {
    updateSettings({
      columns: columns.map((c) => c.dataIndex),
      columnWidths: columns.map((c) => c.defaultWidth),
    });
  }, [ columns.length ]);

  const total = trials.data.length;
  return (
    <InteractiveTable<V1AugmentedTrial>
      columns={columns}
      containerRef={containerRef}
      dataSource={trials.data}
      pagination={getFullPaginationConfig({
        limit: settings.tableLimit,
        offset: settings.tableOffset,
        total,
      }, total)}
      rowClassName={highlights.rowClassName}
      rowKey="trialId"
      rowSelection={{
        getCheckboxProps: () => ({ disabled: A.selectAllMatching }),
        onChange: A.selectTrial,
        preserveSelectedRowKeys: true,
        selectedRowKeys: (A.selectAllMatching ? trials.ids : A.selectedTrials) as number[],
      }}
      scroll={{ x: 1000 }}
      settings={settings}
      showSorterTooltip={false}
      size="small"
      sortDirections={[ 'ascend', 'descend', 'ascend' ]}
      updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      onRow={highlights.onRow}
    />
  );
};

export default TrialTable;
