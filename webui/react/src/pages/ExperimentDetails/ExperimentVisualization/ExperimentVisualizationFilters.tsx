import _ from 'lodash';
import React, { useCallback, useEffect, useMemo } from 'react';

import HpSelect from 'components/HpSelect';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import Select, { Option, SelectValue } from 'components/kit/Select';
import MetricSelect from 'components/MetricSelect';
import RadioGroup from 'components/RadioGroup';
import ScaleSelect from 'components/ScaleSelect';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import { Metric, Scale, ValueOf } from 'types';

import css from './ExperimentVisualizationFilters.module.scss';

export interface VisualizationFilters {
  batch?: number;
  batchMargin?: number;
  hParams: string[];
  maxTrial?: number;
  metric?: Metric;
  scale: Scale;
  view?: ViewType;
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
  batches?: number[];
  filters: VisualizationFilters;
  fullHParams: string[];
  metrics: Metric[];
  onChange?: (filters: Partial<VisualizationFilters>) => void;
  onReset?: () => void;
  type: ExperimentVisualizationType;
}

const TOP_TRIALS_OPTIONS = [1, 10, 20, 50, 100];
const BATCH_MARGIN_OPTIONS = [1, 5, 10, 20, 50];

export const MAX_HPARAM_COUNT = 10;

const ExperimentVisualizationFilters: React.FC<Props> = ({
  batches,
  filters,
  fullHParams,
  metrics,
  onChange,
  onReset,
  type,
}: Props) => {
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

  const handleBatchChange = useCallback(
    (batch: SelectValue) => {
      onChange?.({ batch: batch as number });
    },
    [onChange],
  );

  const handleBatchMarginChange = useCallback(
    (margin: SelectValue) => {
      onChange?.({ batchMargin: margin as number });
    },
    [onChange],
  );

  const handleHParamChange = useCallback(
    (hParams?: SelectValue) => {
      if (!hParams || (Array.isArray(hParams) && hParams.length === 0)) {
        onChange?.({ hParams: fullHParams.slice(0, MAX_HPARAM_COUNT) });
      } else {
        onChange?.({ hParams: hParams as string[] });
      }
    },
    [fullHParams, onChange],
  );

  const handleMaxTrialsChange = useCallback(
    (count: SelectValue) => {
      onChange?.({ maxTrial: count as number });
    },
    [onChange],
  );

  const handleMetricChange = useCallback(
    (metric: Metric) => {
      onChange?.({ metric });
    },
    [onChange],
  );

  const handleViewChange = useCallback(
    (view: SelectValue) => {
      onChange?.({ view: view as ViewType });
    },
    [onChange],
  );

  const handleScaleChange = useCallback(
    (scale: Scale) => {
      onChange?.({ scale });
    },
    [onChange],
  );

  const handleReset = useCallback(() => {
    onReset?.();
  }, [onReset]);

  // Pick the first valid option if the current local batch is invalid.
  useEffect(() => {
    if (!batches || batches.length === 0 || (filters.batch && batches.includes(filters.batch)))
      return;
    onChange?.({ batch: batches.first() });
  }, [batches, filters.batch, onChange]);

  // Pick the first valid option if the current local metric is invalid.
  useEffect(() => {
    if (
      metrics.length === 0 ||
      (!!filters.metric && metrics.some((metric) => _.isEqual(metric, filters.metric)))
    )
      return;
    onChange?.({ metric: metrics.first() });
  }, [filters.metric, metrics, onChange]);

  return (
    <>
      {showMaxTrials && (
        <Select
          dropdownMatchSelectWidth={100}
          label="Top Trials"
          searchable={false}
          value={filters.maxTrial}
          onChange={handleMaxTrialsChange}>
          {TOP_TRIALS_OPTIONS.map((option) => (
            <Option key={option} value={option}>
              {option}
            </Option>
          ))}
        </Select>
      )}
      {showBatches && batches && (
        <>
          <Select
            label="Batches Processed"
            searchable={false}
            value={filters.batch}
            width={70}
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
            value={filters.batchMargin}
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
          value={filters.hParams}
          onChange={handleHParamChange}
        />
      )}
      {showMetrics && (
        <MetricSelect
          defaultMetrics={metrics}
          label="Metric"
          metrics={metrics}
          multiple={false}
          value={filters.metric}
          width={250}
          onChange={handleMetricChange}
        />
      )}
      {showScales && <ScaleSelect value={filters.scale} onChange={handleScaleChange} />}
      {showViews && (
        <RadioGroup
          iconOnly
          options={[
            { icon: 'grid', id: ViewType.Grid, label: 'Table View' },
            { icon: 'list', id: ViewType.List, label: 'Wrapped View' },
          ]}
          value={filters.view ?? ViewType.Grid}
          onChange={handleViewChange}
        />
      )}
      <div className={css.buttons}>
        <Button onClick={handleReset}>
          <Icon name="reset" showTooltip title="Reset" />
        </Button>
      </div>
    </>
  );
};

export default ExperimentVisualizationFilters;
