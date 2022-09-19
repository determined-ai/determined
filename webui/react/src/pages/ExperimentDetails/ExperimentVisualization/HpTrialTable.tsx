import { Typography } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import ResponsiveTable from 'components/ResponsiveTable';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import { paths } from 'routes/utils';
import { Primitive, RecordKey } from 'shared/types';
import { ColorScale, glasbeyColor, rgba2str, rgbaFromGradient,
  str2rgba } from 'shared/utils/color';
import { isNumber } from 'shared/utils/data';
import { alphaNumericSorter, numericSorter, primitiveSorter } from 'shared/utils/sort';
import {
  HyperparametersFlattened, HyperparameterType, MetricName,
} from 'types';

import css from './HpTrialTable.module.scss';

interface Props {
  colorScale?: ColorScale[];
  experimentId: number;
  filteredTrialIdMap?: Record<number, boolean>;
  handleTableRowSelect?: (rowKeys: unknown) => void;
  highlightedTrialId?: number;
  hyperparameters: HyperparametersFlattened;
  metric: MetricName;
  onMouseEnter?: (event: React.MouseEvent, record: TrialHParams) => void;
  onMouseLeave?: (event: React.MouseEvent, record: TrialHParams) => void;
  selectedRowKeys?: number[];
  selection?: boolean;
  trialHps: TrialHParams[];
  trialIds: number[];
}

export interface TrialHParams {
  hparams: Record<RecordKey, Primitive>;
  id: number;
  metric: number | null;
}

const HpTrialTable: React.FC<Props> = ({
  colorScale,
  filteredTrialIdMap,
  hyperparameters,
  highlightedTrialId,
  metric,
  onMouseEnter,
  onMouseLeave,
  trialHps,
  trialIds,
  experimentId,
  selection,
  handleTableRowSelect,
  selectedRowKeys,
}: Props) => {
  const [ pageSize, setPageSize ] = useState(MINIMUM_PAGE_SIZE);

  const dataSource = useMemo(() => {
    if (!filteredTrialIdMap) return trialHps;
    return trialHps.filter((trial) => filteredTrialIdMap[trial.id]);
  }, [ filteredTrialIdMap, trialHps ]);

  const columns = useMemo(() => {
    const idRenderer = (_: string, record: TrialHParams) => {
      const index = trialIds.findIndex((trialId) => trialId === record.id);
      let color = index !== -1 ? glasbeyColor(index) : 'rgba(0, 0, 0, 1.0)';
      if (record.metric != null && colorScale) {
        const scaleRange = colorScale[1].scale - colorScale[0].scale;
        const distance = (record.metric - colorScale[0].scale) / scaleRange;
        const rgbaMin = str2rgba(colorScale[0].color);
        const rgbaMax = str2rgba(colorScale[1].color);
        color = rgba2str(rgbaFromGradient(rgbaMin, rgbaMax, distance));
      }
      return (
        <div className={css.idLayout}>
          <div className={css.colorLegend} style={{ backgroundColor: color }} />
          <Link path={paths.trialDetails(record.id, experimentId)}>
            <div>{record.id}</div>
          </Link>
        </div>
      );
    };
    const idSorter = (a: TrialHParams, b: TrialHParams): number => alphaNumericSorter(a.id, b.id);
    const idColumn = { key: 'id', render: idRenderer, sorter: idSorter, title: 'Trial ID' };

    const metricRenderer = (_: string, record: TrialHParams) => {
      return <HumanReadableNumber num={record.metric} />;
    };
    const metricSorter = (recordA: TrialHParams, recordB: TrialHParams): number => {
      return numericSorter(recordA.metric ?? undefined, recordB.metric ?? undefined);
    };
    const metricColumn = {
      dataIndex: 'metric',
      key: 'metric',
      render: metricRenderer,
      sorter: metricSorter,
      title: <MetricBadgeTag metric={metric} />,
    };

    const hpRenderer = (key: string) => {
      return (_: string, record: TrialHParams) => {
        const value = record.hparams[key];
        const type = hyperparameters[key].type;
        const isValidType = [
          HyperparameterType.Constant,
          HyperparameterType.Double,
          HyperparameterType.Int,
          HyperparameterType.Log,
        ].includes(type);
        if (isNumber(value) && isValidType) {
          return <HumanReadableNumber num={value} />;
        }
        return (
          <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
            {JSON.stringify(value)}
          </Typography.Paragraph>
        );
      };
    };
    const hpColumnSorter = (key: string) => {
      return (recordA: TrialHParams, recordB: TrialHParams): number => {
        const a = recordA.hparams[key] as Primitive;
        const b = recordB.hparams[key] as Primitive;
        return primitiveSorter(a, b);
      };
    };
    const hpColumns = Object
      .keys(hyperparameters || {})
      .map((key) => {
        return {
          key,
          render: hpRenderer(key),
          sorter: hpColumnSorter(key),
          title: key,
        };
      });

    return [ idColumn, metricColumn, ...hpColumns ];
  }, [ colorScale, hyperparameters, metric, trialIds, experimentId ]);

  const handleTableChange = useCallback((tablePagination) => {
    setPageSize(tablePagination.pageSize);
  }, []);

  const handleTableRow = useCallback((record: TrialHParams) => ({
    onMouseEnter: (event: React.MouseEvent) => {
      if (onMouseEnter) onMouseEnter(event, record);
    },
    onMouseLeave: (event: React.MouseEvent) => {
      if (onMouseLeave) onMouseLeave(event, record);
    },
  }), [ onMouseEnter, onMouseLeave ]);

  const rowClassName = useCallback((record: TrialHParams) => {
    return defaultRowClassName({
      clickable: false,
      highlighted: record.id === highlightedTrialId,
    });
  }, [ highlightedTrialId ]);

  return (
    <ResponsiveTable<TrialHParams>
      className={css.base}
      columns={columns}
      dataSource={dataSource}
      pagination={getPaginationConfig(dataSource.length, pageSize)}
      rowClassName={rowClassName}
      rowKey="id"
      rowSelection={selection ? {
        onChange: handleTableRowSelect,
        preserveSelectedRowKeys: true,
        selectedRowKeys,
      } : undefined}
      scroll={{ x: 1000 }}
      showSorterTooltip={false}
      size="small"
      onChange={handleTableChange}
      onRow={handleTableRow}
    />
  );
};

export default HpTrialTable;
