import { Button, Dropdown } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { MutableRefObject, useEffect, useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import BadgeTag from 'components/BadgeTag';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import InteractiveTable, { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import {
  getFullPaginationConfig,
  getPaginationConfig,
  relativeTimeRenderer,
} from 'components/Table/Table';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import UserAvatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import { Highlights } from 'hooks/useHighlight';
import { SettingsHook, UpdateSettings } from 'hooks/useSettings';
import { TrialsWithMetadata } from 'pages/TrialsComparison/Trials/data';
import { paths } from 'routes/utils';
import { Determinedtrialv1State, V1AugmentedTrial } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
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

const getWidthForField = (field: string) => field.length * 8 + 80;

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

  const { filters, setFilters } = C;

  const idColumn = useMemo(() => ({
    dataIndex: 'id',
    defaultWidth: 80,
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
    [ filters.experimentIds, setFilters ],
  );

  const expRankColumn = useMemo(
    () => ({
      dataIndex: 'rank',
      defaultWidth: 140,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterSearch
          {...filterProps}
          value={filters.ranker?.rank || ''}
          onReset={() =>
            setFilters?.((filters) => ({
              ...filters,
              // TODO handle invalid type assertion below
              ranker: { rank: '', sorter: filters.ranker?.sorter as TrialSorter },
            }))
          }
          onSearch={(r) =>
            setFilters?.((filters) => ({
              ...filters,
              // TODO handle invalid type assertion below
              ranker: { rank: r, sorter: filters.ranker?.sorter as TrialSorter },
            }))
          }
        />
      ),
      isFiltered: () => !!filters.ranker?.rank,
      key: 'rank',
      render: (_: string, record: V1AugmentedTrial) => (
        <div className={css.idLayout}>{record.rankWithinExp}</div>
      ),
      sorter: true,
      title: 'Rank in Exp',
    }),
    [ filters.ranker?.rank, setFilters ],
  );

  const hpColumns = useMemo(() => Object
    .keys(trials.hparams || {})
    // .filter((hp) => trials.hparams[hp]?.size > 1)
    .map((hp) => {
      return {
        dataIndex: hp,
        defaultWidth: getWidthForField(hp),
        filterDropdown: rangeFilterForPrefix('hparams', filters, setFilters)(hp),
        isFiltered: () => rangeFilterIsActive(filters.hparams, hp),
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
    }), [ filters, trials.hparams, setFilters ]);

  const tagColumn = useMemo(() => ({
    dataIndex: 'tags',
    defaultWidth: 70,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
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
    key: 'labels',
    render: trialTagsRenderer,
    sorter: false,
    title: 'Tags',
  }), [ filters.tags, setFilters ]);

  const trainingMetricColumns = useMemo(() => trials.metrics
    .filter((metric) => metric.type = MetricType.Training).map((metric) => {
      const key = metricToKey(metric);
      return {
        dataIndex: key,
        defaultWidth: getWidthForField(metric.name),
        filterDropdown: rangeFilterForPrefix(
          'trainingMetrics',
          filters,
          setFilters,
        )(metric.name),
        isFiltered: () => rangeFilterIsActive(filters.trainingMetrics, metric.name),
        key: `trainingMetrics.${metric.name}`,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.trainingMetrics?.[metricKeyToName(key)];
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,

      };
    }), [ filters, trials.metrics, setFilters ]);

  const validationMetricColumns = useMemo(() => trials.metrics
    .filter((metric) => metric.type = MetricType.Validation).map((metric) => {
      const key = metricToKey(metric);
      return {
        dataIndex: key,
        defaultWidth: getWidthForField(metric.name),
        filterDropdown: rangeFilterForPrefix(
          'validationMetrics',
          filters,
          setFilters,
        )(metric.name),
        isFiltered: () => rangeFilterIsActive(filters.validationMetrics, metric.name),
        key: `validationMetrics.${metric.name}`,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.validationMetrics?.[metricKeyToName(key)];
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,
      };
    }), [ filters, trials.metrics, setFilters ]);

  const stateColumn = useMemo(() => ({
    dataIndex: 'state',
    defaultWidth: 110,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={(filters.states ?? [])}
        onFilter={(states) => setFilters?.((filters) => ({ ...filters, states }))}
        onReset={() => setFilters?.((filters) => ({ ...filters, states: undefined }))}
      />
    ),
    filters: [
      RunState.Active,
      RunState.Paused,
      RunState.Canceled,
      RunState.Completed,
      RunState.Errored,
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
  }), [ setFilters, filters ]);

  const startTimeColumn = useMemo(() => ({
    dataIndex: 'startTime',
    defaultWidth: 110,
    key: 'startTime',
    render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  }), []);

  const endTimeColumn = useMemo(() => ({
    dataIndex: 'startTime',
    defaultWidth: 110,
    key: 'startTime',
    render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  }), []);

  const experimentNameColumn = useMemo(() => ({
    dataIndex: 'experimentName',
    defaultWidth: 160,
    key: 'experimentName',
    render: (value: string | number | undefined, record: V1AugmentedTrial): React.ReactNode => (
      <Link path={paths.experimentDetails(record.experimentId)}>
        {value === undefined ? '' : value}
      </Link>
    ),
    title: 'Experiment Name',
  }), []);

  const searcherTypeColumn = useMemo(() => ({
    dataIndex: 'searcherType',
    defaultWidth: 120,
    key: 'searcherType',
    title: 'Searcher Type',
  }), []);

  const searcherMetricColumn = useMemo(() => ({
    dataIndex: 'searcherMetric',
    defaultWidth: 120,
    key: 'searcherMetric',
    sorter: true,
    title: (
      <BadgeTag
        label="Metric"
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
        values={filters.userIds}
        onFilter={(userIds: string[]) => setFilters?.((filters) => ({ ...filters, userIds }))}
        onReset={() => setFilters?.((filters) => ({ ...filters, userIds: undefined }))}
      />
    ),
    filters: users.map((user) => ({ text: getDisplayName(user), value: user.username })),
    isFiltered: () => !!filters.userIds,
    key: 'userId',
    render: (_: string, record: V1AugmentedTrial) => <UserAvatar userId={record.userId} />,
    sorter: true,
    title: 'User',
  }), [ filters.userIds, setFilters, users ]);

  const totalBatchesColumn = useMemo(() => ({
    dataIndex: 'totalBatches',
    defaultWidth: 120,
    key: 'totalBatches',
    sorter: true,
    title: 'Total Batches',
  }), []);

  const searcherMetricValueColumn = useMemo(() => ({
    dataIndex: 'searcherMetricValue',
    defaultWidth: 100,
    key: 'searcherMetricValue',
    render: (_: string, record: V1AugmentedTrial) => (
      <HumanReadableNumber num={record.searcherMetricValue} />
    ),
    sorter: true,
    title: (
      <BadgeTag
        label="Value"
        tooltip="Searcher Metric Value">
        S
      </BadgeTag>
    ),
  }), []);

  const actionColumn = useMemo(() => ({
    align: 'right',
    className: 'fullCell',
    dataIndex: 'action',
    defaultWidth: 80,
    fixed: 'right',
    key: 'action',
    render: (_:string, record: V1AugmentedTrial) => (
      <Dropdown
        overlay={() => <></>}
        trigger={[ 'click' ]}>
        <Button className={css.overflow} type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
    ),
    title: '',
    width: 80,
  }), []);

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
    actionColumn,
  ].map((col) => ({ defaultWidth: 80, ...col })), [
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

  useEffect(() => {
    console.log('CCC', columns.map((c) => c.dataIndex));
    updateSettings({
      columns: columns.map((c) => c.dataIndex).slice(0, -1),
      columnWidths: columns.map((c) => c.defaultWidth ?? 80),
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
