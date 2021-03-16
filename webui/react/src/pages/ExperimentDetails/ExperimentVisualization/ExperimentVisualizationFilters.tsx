import { Button, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo, useReducer } from 'react';

import IconButton from 'components/IconButton';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import SelectFilter from 'components/SelectFilter';
import { ExperimentVisualizationType, MetricName } from 'types';

import css from './ExperimentVisualizationFilters.module.scss';

const { Option } = Select;

export interface VisualizationFilters {
  batch: number;
  batchMargin: number;
  hParams: string[];
  maxTrial: number;
  metric: MetricName;
}

export enum FilterError {
  MetricBatches,
  MetricNames,
}

interface Props {
  batches: number[];
  filters: VisualizationFilters;
  fullHParams: string[];
  metrics: MetricName[];
  onChange?: (filters: VisualizationFilters) => void;
  onMetricChange?: (metric: MetricName) => void;
  type: ExperimentVisualizationType,
}

enum ActionType { Set, SetBatch, SetBatchMargin, SetHParams, SetMaxTrial, SetMetric }

type Action =
| { type: ActionType.Set; value: VisualizationFilters }
| { type: ActionType.SetBatch; value: number }
| { type: ActionType.SetBatchMargin; value: number }
| { type: ActionType.SetHParams; value: string[] }
| { type: ActionType.SetMaxTrial; value: number }
| { type: ActionType.SetMetric; value: MetricName }

const TOP_TRIALS_OPTIONS = [ 1, 10, 20, 50, 100 ];
const BATCH_MARGIN_OPTIONS = [ 1, 5, 10, 20, 50 ];

export const MAX_HPARAM_COUNT = 10;

const reducer = (state: VisualizationFilters, action: Action) => {
  switch (action.type) {
    case ActionType.Set:
      return { ...action.value };
    case ActionType.SetBatch:
      return { ...state, batch: action.value };
    case ActionType.SetBatchMargin:
      return { ...state, batchMargin: action.value };
    case ActionType.SetHParams:
      return { ...state, hParams: action.value };
    case ActionType.SetMaxTrial:
      return { ...state, maxTrial: action.value };
    case ActionType.SetMetric:
      return { ...state, metric: action.value };
    default:
      return state;
  }
};

const ExperimentVisualizationFilters: React.FC<Props> = ({
  batches,
  filters,
  fullHParams,
  metrics,
  onChange,
  onMetricChange,
  type,
}: Props) => {
  const [ localFilters, dispatch ] = useReducer(reducer, filters);

  const [ showMaxTrials, showBatches, showMetrics, showHParams ] = useMemo(() => {
    return [
      [ ExperimentVisualizationType.LearningCurve ].includes(type),
      [
        ExperimentVisualizationType.HpHeatMap,
        ExperimentVisualizationType.HpParallelCoordinates,
        ExperimentVisualizationType.HpScatterPlots,
      ].includes(type),
      [
        ExperimentVisualizationType.HpHeatMap,
        ExperimentVisualizationType.HpParallelCoordinates,
        ExperimentVisualizationType.HpScatterPlots,
        ExperimentVisualizationType.LearningCurve,
      ].includes(type),
      [
        ExperimentVisualizationType.HpHeatMap,
        ExperimentVisualizationType.HpParallelCoordinates,
        ExperimentVisualizationType.HpScatterPlots,
      ].includes(type),
      [ ExperimentVisualizationType.HpHeatMap ].includes(type),
    ];
  }, [ type ]);

  const handleBatchChange = useCallback((batch: SelectValue) => {
    dispatch({ type: ActionType.SetBatch, value: batch as number });
  }, []);

  const handleBatchMarginChange = useCallback((margin: SelectValue) => {
    dispatch({ type: ActionType.SetBatchMargin, value: margin as number });
  }, []);

  const handleHParamChange = useCallback((hParams?: SelectValue) => {
    if (!hParams || (Array.isArray(hParams) && hParams.length === 0)) {
      dispatch({ type: ActionType.SetHParams, value: fullHParams.slice(0, MAX_HPARAM_COUNT) });
    } else {
      dispatch({ type: ActionType.SetHParams, value: hParams as string[] });
    }
  }, [ fullHParams ]);

  const handleMaxTrialsChange = useCallback((count: SelectValue) => {
    dispatch({ type: ActionType.SetMaxTrial, value: count as number });
  }, []);

  const handleMetricChange = useCallback((metric: MetricName) => {
    dispatch({ type: ActionType.SetMetric, value: metric });
    if (onMetricChange) onMetricChange(metric);
  }, [ onMetricChange ]);

  const handleApply = useCallback(() => {
    if (onChange) onChange(localFilters);
  }, [ localFilters, onChange ]);

  const handleReset = useCallback(() => {
    dispatch({ type: ActionType.Set, value: filters });
  }, [ filters ]);

  return (
    <div className={css.base}>
      {showMaxTrials && (
        <SelectFilter
          enableSearchFilter={false}
          label="Top Trials"
          showSearch={false}
          style={{ width: 70 }}
          value={localFilters.maxTrial}
          onChange={handleMaxTrialsChange}>
          {TOP_TRIALS_OPTIONS.map(option => <Option key={option} value={option}>{option}</Option>)}
        </SelectFilter>
      )}
      {showBatches && (
        <>
          <SelectFilter
            enableSearchFilter={false}
            label="Batches Processed"
            showSearch={false}
            value={localFilters.batch}
            onChange={handleBatchChange}>
            {batches.map(batch => <Option key={batch} value={batch}>{batch}</Option>)}
          </SelectFilter>
          <SelectFilter
            enableSearchFilter={false}
            label="Batch Margin"
            showSearch={false}
            value={localFilters.batchMargin}
            onChange={handleBatchMarginChange}>
            {BATCH_MARGIN_OPTIONS.map(option => (
              <Option key={option} value={option}>{option}</Option>
            ))}
          </SelectFilter>
        </>
      )}
      {showMetrics && (
        <MetricSelectFilter
          defaultMetricNames={metrics}
          label="Metric"
          metricNames={metrics}
          multiple={false}
          value={localFilters.metric}
          width={'100%'}
          onChange={handleMetricChange} />
      )}
      {showHParams && (
        <MultiSelect
          label="HP"
          value={localFilters.hParams}
          onChange={handleHParamChange}>
          {fullHParams.map(hParam => <Option key={hParam} value={hParam}>{hParam}</Option>)}
        </MultiSelect>
      )}
      <div className={css.buttons}>
        <IconButton icon="checkmark" label="Apply Filters" type="primary" onClick={handleApply} />
        <IconButton icon="close" label="Reset Filters" onClick={handleReset} />
      </div>
    </div>
  );
};

export default ExperimentVisualizationFilters;
