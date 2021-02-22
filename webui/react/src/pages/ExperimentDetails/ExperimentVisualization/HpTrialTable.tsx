import React, { useCallback, useMemo, useState } from 'react';

import BadgeTag from 'components/BadgeTag';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Link from 'components/Link';
import ResponsiveTable from 'components/ResponsiveTable';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import { paths } from 'routes/utils';
import { ExperimentHyperParams, ExperimentHyperParamType, MetricName, Primitive } from 'types';
import { ColorScale, glasbeyColor, rgba2str, rgbaFromGradient, str2rgba } from 'utils/color';
import { alphanumericSorter, isNumber, numericSorter, primitiveSorter } from 'utils/data';

import css from './HpTrialTable.module.scss';

type HParams = Record<string, Primitive>;

interface Props {
  colorScale?: ColorScale[];
  experimentId: number;
  filteredTrialIdMap?: Record<number, boolean>;
  highlightedTrialId?: number;
  hyperparameters: ExperimentHyperParams;
  metric: MetricName;
  onMouseEnter?: (event: React.MouseEvent, record: TrialHParams) => void;
  onMouseLeave?: (event: React.MouseEvent, record: TrialHParams) => void;
  trialHps: TrialHParams[];
  trialIds: number[];
}

export interface TrialHParams {
  hparams: HParams;
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
}: Props) => {
  const [ pageSize, setPageSize ] = useState(MINIMUM_PAGE_SIZE);

  const dataSource = useMemo(() => {
    if (!filteredTrialIdMap) return trialHps;
    return trialHps.filter(trial => filteredTrialIdMap[trial.id]);
  }, [ filteredTrialIdMap, trialHps ]);

  const columns = useMemo(() => {
    const idRenderer = (_: string, record: TrialHParams) => {
      const index = trialIds.findIndex(trialId => trialId === record.id);
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
    const idSorter = (a: TrialHParams, b: TrialHParams): number => alphanumericSorter(a.id, b.id);
    const idColumn = { key: 'id', render: idRenderer, sorter: idSorter, title: 'Trial ID' };

    const metricRenderer = (_: string, record: TrialHParams) => {
      return record.metric ? <HumanReadableFloat num={record.metric} /> : null;
    };
    const metricSorter = (recordA: TrialHParams, recordB: TrialHParams): number => {
      return numericSorter(recordA.metric || undefined, recordB.metric || undefined);
    };
    const metricColumn = {
      dataIndex: 'metric',
      key: 'metric',
      render: metricRenderer,
      sorter: metricSorter,
      title: <BadgeTag
        label={metric.name}
        tooltip={metric.type}>{metric.type.substr(0, 1).toUpperCase()}</BadgeTag>,
    };

    const hpRenderer = (key: string) => {
      return (_: string, record: TrialHParams) => {
        const value = record.hparams[key];
        const type = hyperparameters[key].type;
        const isValidType = [
          ExperimentHyperParamType.Constant,
          ExperimentHyperParamType.Double,
          ExperimentHyperParamType.Int,
          ExperimentHyperParamType.Log,
        ].includes(type);
        if (isNumber(value) && isValidType) {
          return <HumanReadableFloat num={value} />;
        }
        return record.hparams[key];
      };
    };
    const hpColumnSorter = (key: string) => {
      return (recordA: TrialHParams, recordB: TrialHParams): number => {
        const a = recordA.hparams[key];
        const b = recordB.hparams[key];
        return primitiveSorter(a, b);
      };
    };
    const hpColumns = Object
      .keys(hyperparameters || {})
      .map(key => {
        return {
          key,
          render: hpRenderer(key),
          sorter: hpColumnSorter(key),
          title: key,
        };
      });

    return [ idColumn, metricColumn, ...hpColumns ];
  }, [ colorScale, hyperparameters, metric, trialIds ]);

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
      columns={columns}
      dataSource={dataSource}
      pagination={getPaginationConfig(dataSource.length, pageSize)}
      rowClassName={rowClassName}
      rowKey="id"
      scroll={{ x: 1000 }}
      showSorterTooltip={false}
      size="small"
      onChange={handleTableChange}
      onRow={handleTableRow} />

  );
};

export default HpTrialTable;
