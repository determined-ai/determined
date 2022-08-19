import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { MutableRefObject, useEffect, useMemo } from 'react';

import BadgeTag from 'components/BadgeTag';
import HumanReadableNumber from 'components/HumanReadableNumber';
import InteractiveTable, { InteractiveTableSettings } from 'components/InteractiveTable';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import { getPaginationConfig } from 'components/Table';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
import { Highlights } from 'hooks/useHighlight';
import { SettingsHook, UpdateSettings } from 'hooks/useSettings';
import { TrialsWithMetadata } from 'pages/TrialsComparison/Trials/data';
import { paths } from 'routes/utils';
import { V1AugmentedTrial } from 'services/api-ts-sdk';
import { ColorScale, glasbeyColor } from 'shared/utils/color';
import { isNumber } from 'shared/utils/data';
import { MetricType } from 'types';
import { metricKeyToName, metricToKey } from 'utils/metric';

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

const hpTitle = (hp: string) => <BadgeTag label={hp} tooltip="Hyperparameter">H</BadgeTag>;

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

  const idColumn = useMemo(() => ({
    dataIndex: 'id',
    defaultWidth: 60,
    key: 'id',
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
      title: 'Rank in Exp',
    }),
    [ C.filters.ranker?.rank, C.setFilters ],
  );

  const hpColumns = useMemo(() => Object
    .keys(trials.hparams || {})
    .filter((hp) => trials.hparams[hp]?.size > 1)
    .map((key) => {
      return {
        dataIndex: key,
        defaultWidth: 130,
        filterDropdown: rangeFilterForPrefix('hparams', C.filters, C.setFilters)(key),
        isFiltered: () => rangeFilterIsActive(C.filters, 'hparams', key),
        key,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.hparams[key];
          if (isNumber(value)) {
            return <HumanReadableNumber num={value} />;
          } else if (!value) {
            return '-';
          }
          return value + '';
        },
        sorter: true,
        title: hpTitle(key),
      };
    }), [ C.filters, trials.hparams, C.setFilters ]);

  const tagColumn = useMemo(() => ({
    dataIndex: 'tags',
    defaultWidth: 60,
    filterDropdown: (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        validatorRegex={/[^a-zA-Z0-9]+$/} // TODO need fix ?
        values={C.filters.tags}
        onReset={() => C.setFilters?.((filters) => ({ ...filters, tags: [] }))}
      />
    ),
    isFiltered: () => !!C.filters.tags?.length,
    key: 'labels',
    render: trialTagsRenderer,
    sorter: true,
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
        key,
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
        key,
        render: (_: string, record: V1AugmentedTrial) => {
          const value = record.validationMetrics?.[metricKeyToName(key)];
          return isNumber(value) ? <HumanReadableNumber num={value} /> : '-';
        },
        sorter: true,
        title: <MetricBadgeTag metric={metric} />,
      };
    }), [ C.filters, trials.metrics, C.setFilters ]);

  // console.log(metrics);

  const columns = useMemo(() => [
    idColumn,
    experimentIdColumn,
    expRankColumn,
    tagColumn,
    ...hpColumns,
    ...trainingMetricColumns,
    ...validationMetricColumns,
  ], [
    idColumn,
    experimentIdColumn,
    expRankColumn,
    tagColumn,
    hpColumns,
    trainingMetricColumns,
    validationMetricColumns,
  ]);

  useEffect(() => {

    // updateSettings({
    //   columns: columns.map((c) => c.dataIndex),
    //   columnWidths: columns.map((c) => c.defaultWidth),
    // });
  }, [ columns.length ]);

  return (
    <InteractiveTable<V1AugmentedTrial>
      columns={columns}
      containerRef={containerRef}
      dataSource={trials.data}
      pagination={getPaginationConfig(trials.data.length, 10)}
      rowClassName={highlights.rowClassName}
      rowKey="trialId"
      rowSelection={A.selectedTrials.length ? {
        getCheckboxProps: () => ({ disabled: A.selectAllMatching }),
        onChange: A.selectTrial,
        preserveSelectedRowKeys: true,
        selectedRowKeys: (A.selectAllMatching ? trials.ids : A.selectedTrials) as number[],
      } : undefined}
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
