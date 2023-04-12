import { Tooltip } from 'antd';
import React, { useCallback, useEffect, useMemo, useReducer } from 'react';

import HpSelect from 'components/HpSelect';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import Select, { Option, SelectValue } from 'components/kit/Select';
import MetricSelect from 'components/MetricSelect';
import RadioGroup from 'components/RadioGroup';
import ScaleSelect from 'components/ScaleSelect';
import { ValueOf } from 'shared/types';
import { Metric, Scale } from 'types';

import { ExperimentVisualizationType } from '../ExperimentVisualization';

import css from './ExperimentVisualizationFilters.module.scss';

export interface VisualizationFilters {
  batch: number;
  batchMargin: number;
  hParams: string[];
  maxTrial: number;
  metric: Metric | null;
  scale: Scale;
  view: ViewType;
}

export const FilterError = {
  MetricBatches: 'MetricBatches',
  Metrics: 'Metrics',
} as const;

export type FilterError = ValueOf<typeof FilterError>;

export const ViewType = {
  Grid: 'grid',
  List: 'list',
} as const;

export type ViewType = ValueOf<typeof ViewType>;

interface Props {
  batches: number[];
  filters: VisualizationFilters;
  fullHParams: string[];
  metrics: Metric[];
  onChange?: (filters: VisualizationFilters) => void;
  onMetricChange?: (metric: Metric) => void;
  onReset?: () => void;
  type: ExperimentVisualizationType;
}

const ActionType = {
  Set: 0,
  SetBatch: 1,
  SetBatchMargin: 2,
  SetHParams: 3,
  SetMaxTrial: 4,
  SetMetric: 5,
  SetScale: 6,
  SetView: 7,
} as const;

type ActionType = ValueOf<typeof ActionType>;

type Action =
  | { type: typeof ActionType.Set; value: VisualizationFilters }
  | { type: typeof ActionType.SetBatch; value: number }
  | { type: typeof ActionType.SetBatchMargin; value: number }
  | { type: typeof ActionType.SetHParams; value: string[] }
  | { type: typeof ActionType.SetMaxTrial; value: number }
  | { type: typeof ActionType.SetMetric; value: Metric }
  | { type: typeof ActionType.SetView; value: ViewType }
  | { type: typeof ActionType.SetScale; value: Scale };

const TOP_TRIALS_OPTIONS = [1, 10, 20, 50, 100];
const BATCH_MARGIN_OPTIONS = [1, 5, 10, 20, 50];

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
    case ActionType.SetView:
      return { ...state, view: action.value };
    case ActionType.SetScale:
      return { ...state, scale: action.value };
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
  onReset,
  type,
}: Props) => {
  const [localFilters, dispatch] = useReducer(reducer, filters);

  const [showMaxTrials, showBatches, showMetrics, showHParams, showViews, showScales] =
    useMemo(() => {
      return [
        ExperimentVisualizationType.LearningCurve === type,
        ExperimentVisualizationType.HpHeatMap === type ||
          ExperimentVisualizationType.HpParallelCoordinates === type ||
          ExperimentVisualizationType.HpScatterPlots === type,
        [
          ExperimentVisualizationType.HpHeatMap,
          ExperimentVisualizationType.HpParallelCoordinates,
          ExperimentVisualizationType.HpScatterPlots,
          ExperimentVisualizationType.LearningCurve,
        ].includes(type),
        ExperimentVisualizationType.HpHeatMap === type ||
          ExperimentVisualizationType.HpParallelCoordinates === type ||
          ExperimentVisualizationType.HpScatterPlots === type,
        ExperimentVisualizationType.HpHeatMap === type,
        [
          ExperimentVisualizationType.HpHeatMap,
          ExperimentVisualizationType.HpScatterPlots,
          ExperimentVisualizationType.LearningCurve,
          ExperimentVisualizationType.HpParallelCoordinates,
        ].includes(type),
      ];
    }, [type]);

  const handleBatchChange = useCallback((batch: SelectValue) => {
    dispatch({ type: ActionType.SetBatch, value: batch as number });
  }, []);

  const handleBatchMarginChange = useCallback((margin: SelectValue) => {
    dispatch({ type: ActionType.SetBatchMargin, value: margin as number });
  }, []);

  const handleHParamChange = useCallback(
    (hParams?: SelectValue) => {
      if (!hParams || (Array.isArray(hParams) && hParams.length === 0)) {
        dispatch({ type: ActionType.SetHParams, value: fullHParams.slice(0, MAX_HPARAM_COUNT) });
      } else {
        dispatch({ type: ActionType.SetHParams, value: hParams as string[] });
      }
    },
    [fullHParams],
  );

  const handleMaxTrialsChange = useCallback((count: SelectValue) => {
    dispatch({ type: ActionType.SetMaxTrial, value: count as number });
  }, []);

  const handleMetricChange = useCallback(
    (metric: Metric) => {
      dispatch({ type: ActionType.SetMetric, value: metric });
      if (onMetricChange) onMetricChange(metric);
    },
    [onMetricChange],
  );

  const handleViewChange = useCallback((view: SelectValue) => {
    dispatch({ type: ActionType.SetView, value: view as ViewType });
  }, []);

  const handleScaleChange = useCallback((scale: Scale) => {
    dispatch({ type: ActionType.SetScale, value: scale });
  }, []);

  useEffect(() => {
    if (onChange) onChange(localFilters);
  }, [localFilters, onChange]);

  const handleReset = useCallback(() => {
    dispatch({ type: ActionType.Set, value: filters });
    if (onReset) onReset();
  }, [filters, onReset]);

  // Pick the first valid option if the current local batch is invalid.
  useEffect(() => {
    if (batches.includes(localFilters.batch)) return;
    dispatch({ type: ActionType.SetBatch, value: batches.first() });
  }, [batches, localFilters.batch]);

  return (
    <>
      {showMaxTrials && (
        <Select
          label="Top Trials"
          searchable={false}
          value={localFilters.maxTrial}
          onChange={handleMaxTrialsChange}>
          {TOP_TRIALS_OPTIONS.map((option) => (
            <Option key={option} value={option}>
              {option}
            </Option>
          ))}
        </Select>
      )}
      {showBatches && (
        <>
          <Select
            label="Batches Processed"
            searchable={false}
            value={localFilters.batch}
            onChange={handleBatchChange}>
            {batches.map((batch) => (
              <Option key={batch} value={batch}>
                {batch}
              </Option>
            ))}
          </Select>
          <Select
            label="Batch Margin"
            searchable={false}
            value={localFilters.batchMargin}
            onChange={handleBatchMarginChange}>
            {BATCH_MARGIN_OPTIONS.map((option) => (
              <Option key={option} value={option}>
                {option}
              </Option>
            ))}
          </Select>
        </>
      )}
      {showHParams && (
        <HpSelect
          fullHParams={fullHParams}
          label={`HP (max ${MAX_HPARAM_COUNT})`}
          value={localFilters.hParams}
          onChange={handleHParamChange}
        />
      )}
      {showMetrics && (
        <MetricSelect
          defaultMetrics={metrics}
          label="Metric"
          metrics={metrics}
          multiple={false}
          value={localFilters.metric || undefined}
          width={250}
          onChange={handleMetricChange}
        />
      )}
      {showScales && <ScaleSelect value={localFilters.scale} onChange={handleScaleChange} />}
      {showViews && (
        <RadioGroup
          iconOnly
          options={[
            { icon: 'grid', id: ViewType.Grid, label: 'Table View' },
            { icon: 'list', id: ViewType.List, label: 'Wrapped View' },
          ]}
          value={localFilters.view}
          onChange={handleViewChange}
        />
      )}
      <div className={css.buttons}>
        <Tooltip title="Reset">
          <Button onClick={handleReset}>
            <Icon name="reset" />
          </Button>
        </Tooltip>
      </div>
    </>
  );
};

export default ExperimentVisualizationFilters;
