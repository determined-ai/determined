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

  relativeTimeRenderer,
} from 'components/Table/Table';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterRank from 'components/Table/TableFilterRank';
import UserAvatar from 'components/UserAvatar';
import { useStore } from 'contexts/Store';
import { Highlights } from 'hooks/useHighlight';
import { SettingsHook, UpdateSettings } from 'hooks/useSettings';
import { TrialsWithMetadata } from 'pages/TrialsComparison/Trials/data';
import { paths } from 'routes/utils';
import { Determinedtrialv1State, V1AugmentedTrial, V1OrderBy } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import { ColorScale, glasbeyColor } from 'shared/utils/color';
import { isNumber, numberElseUndefined } from 'shared/utils/data';
import { StateOfUnion } from 'themes';
import { MetricType, RunState, TrialState } from 'types';
import { metricKeyToName, metricToKey } from 'utils/metric';
import { getDisplayName } from 'utils/user';

import { TrialActionsInterface } from '../Actions/useTrialActions';
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
  total: number;
  trialsWithMetadata: TrialsWithMetadata;
}

const getWidthForField = (field: string) => field.length * 8 + 80;
const nonRankableColumns = [
  'experimentId',
  'experimentName',
  'userId',
  'rank',
  'searcherMetric',
  'searcherType',
];

const TrialTable: React.FC<Props> = ({
  // colorScale,
  collectionsInterface: C,
  actionsInterface: A,
  trialsWithMetadata: trials,
  containerRef,
  highlights,
  tableSettingsHook,
  total,
}: Props) => {

  const { settings, updateSettings } = tableSettingsHook;

  const { users } = useStore();

  const { filters, setFilters } = C;

  const idColumn = useMemo(() => ({
    defaultWidth: 100,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        validatorRegex={/\D/}
        values={filters.trialIds}
        onFilter={(trialIds: string[]) =>
          setFilters?.((filters) => ({ ...filters, trialIds }))
        }
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
  }), [ filters.trialIds, setFilters ]);

  const experimentIdColumn = useMemo(
    () => ({
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
      defaultWidth: 130,
      filterDropdown: (filterProps: FilterDropdownProps) => (
        <TableFilterRank
          {...filterProps}
          columns={settings.columns.filter((col) => !nonRankableColumns.includes(col))}
          rank={filters.ranker?.rank}
          onReset={() =>
            setFilters?.((filters) => ({
              ...filters,
              ranker: {
                rank: '',
                sorter: { orderBy: V1OrderBy.ASC, sortKey: 'searcherMetricValue' },
              },
            }))
          }
          onSet={(column, orderBy, rank) =>
            setFilters?.((filters) => ({
              ...filters,
              ranker: { rank, sorter: { orderBy, sortKey: column } },
            }))
          }
        />
      ),
      isFiltered: () => filters.ranker?.rank ? parseInt(filters.ranker?.rank) !== 0 : false,
      key: 'rank',
      render: (_: string, record: V1AugmentedTrial) => (
        <div className={css.idLayout}>{record.rankWithinExp}</div>
      ),
      sorter: true,
      title: 'Rank in Exp',
    }),
    [ filters.ranker?.rank, setFilters, settings.columns ],
  );

  const hpColumns = useMemo(() => Object
    .keys(trials.hparams || {})
    // .filter((hp) => trials.hparams[hp]?.size > 1)
    .map((hp) => {
      const columnKey = `hparams.${hp}`;
      const actionable = [ ...trials.hparams[hp] ].every((val) =>
        Number.isFinite(parseFloat(String(val))));
      return {
        defaultWidth: getWidthForField(hp),
        filterDropdown: actionable
          ? rangeFilterForPrefix('hparams', filters, setFilters)(hp)
          : null,
        isFiltered: () => rangeFilterIsActive(filters.hparams, hp),
        key: columnKey,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.hparams[hp];
          if (isNumber(value)) {
            return <HumanReadableNumber num={value} />;
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
    }), [ filters, trials.hparams, setFilters ]);

  const tagColumn = useMemo(() => ({
    defaultWidth: 90,
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
    key: 'tags',
    render: trialTagsRenderer,
    sorter: false,
    title: 'Tags',
  }), [ filters.tags, setFilters ]);

  const trainingMetricColumns = useMemo(() => trials.metrics
    .filter((metric) => metric.type = MetricType.Training).map((metric) => {
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
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,

      };
    }), [ filters, trials.metrics, setFilters ]);

  const validationMetricColumns = useMemo(() => trials.metrics
    .filter((metric) => metric.type = MetricType.Validation).map((metric) => {
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
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,
      };
    }), [ filters, trials.metrics, setFilters ]);

  const stateColumn = useMemo(() => ({
    defaultWidth: 100,
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
    defaultWidth: 100,
    key: 'startTime',
    render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  }), []);

  const endTimeColumn = useMemo(() => ({
    defaultWidth: 100,
    key: 'endTime',
    render: (_: number, record: V1AugmentedTrial): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'End Time',
  }), []);

  const experimentNameColumn = useMemo(() => ({
    defaultWidth: 145,
    key: 'experimentName',
    render: (value: string | number | undefined, record: V1AugmentedTrial): React.ReactNode => (
      <Link path={paths.experimentDetails(record.experimentId)}>
        {value === undefined ? '' : value}
      </Link>
    ),
    title: 'Experiment Name',
  }), []);

  const searcherTypeColumn = useMemo(() => ({
    defaultWidth: 120,
    key: 'searcherType',
    title: 'Searcher Type',
  }), []);

  const searcherMetricColumn = useMemo(() => ({
    defaultWidth: 125,
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
    isFiltered: () => !!filters.userIds?.length,
    key: 'userId',
    render: (_: string, record: V1AugmentedTrial) => <UserAvatar userId={record.userId} />,
    sorter: true,
    title: 'User',
  }), [ filters.userIds, setFilters, users ]);

  const totalBatchesColumn = useMemo(() => ({
    defaultWidth: 130,
    key: 'totalBatches',
    sorter: true,
    title: 'Total Batches',
  }), []);

  const searcherLossColumn = useMemo(() => ({
    defaultWidth: 110,
    key: 'searcherMetricLoss',
    render: (_: string, record: V1AugmentedTrial) => (
      <HumanReadableNumber num={record.searcherMetricLoss} />
    ),
    sorter: true,
    title: (
      <BadgeTag
        label="Loss"
        tooltip="Searcher Metric Loss">
        S
      </BadgeTag>
    ),
  }), []);

  const actionColumn = useMemo(() => ({
    align: 'right',
    className: 'fullCell',
    defaultWidth: 110,
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
    searcherLossColumn,
    searcherMetricColumn,
    searcherTypeColumn,
    experimentNameColumn,
    stateColumn,
    startTimeColumn,
    endTimeColumn,
    ...hpColumns,
    // ...trainingMetricColumns,
    ...validationMetricColumns,
    actionColumn,
  ].map((col) => ({ ...col, dataIndex: col.key })), [
    actionColumn,
    idColumn,
    experimentIdColumn,
    expRankColumn,
    tagColumn,
    hpColumns,
    // trainingMetricColumns,
    validationMetricColumns,
    userColumn,
    totalBatchesColumn,
    searcherLossColumn,
    searcherMetricColumn,
    searcherTypeColumn,
    stateColumn,
    experimentNameColumn,
    startTimeColumn,
    endTimeColumn,
  ]);

  // console.log(validationMetricColumns);

  useEffect(() => {
    updateSettings({
      columns: columns.map((c) => c.key).slice(0, -1),
      columnWidths: columns.map((c) => c.defaultWidth),
    });
  }, [ columns.length ]);

  const pagination = useMemo(() => {
    const limit = settings.tableLimit;
    const offset = settings.tableOffset;
    // right now we are getting total from API. but this results in an extra
    // (more expensive) query to get the total row count in order to display
    // how many pages there are in the table. this is a hack to make it always
    // display 2 additional pages beyond the current
    const fakeTotal = offset + 2 * limit + 1;
    return getFullPaginationConfig({
      limit,
      offset,
    }, total);
  }, [ settings.tableLimit, settings.tableOffset, total ]);

  return (
    <InteractiveTable<V1AugmentedTrial>
      columns={columns}
      containerRef={containerRef}
      dataSource={trials.data}
      pagination={pagination}
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
