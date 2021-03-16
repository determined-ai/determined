import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import GridListRadioGroup, { GridListView } from 'components/GridListRadioGroup';
import MetricSelectFilter from 'components/MetricSelectFilter';
import MultiSelect from 'components/MultiSelect';
import SelectFilter from 'components/SelectFilter';
import { ExperimentVisualizationType, MetricName } from 'types';

import css from './ExperimentVisualizationFilters.module.scss';

const { Option } = Select;

interface Props {
  batches: number[];
  hParams: string[];
  metrics: MetricName[];
  onBatchChange?: (batch: number) => void;
  onBatchMarginChange?: (margin: number) => void;
  onHParamChange?: (hParams?: string[]) => void;
  onMaxTrialsChange?: (count: number) => void;
  onMetricChange?: (metric: MetricName) => void;
  onViewChange?: (view: GridListView) => void;
  selectedBatch: number;
  selectedBatchMargin?: number;
  selectedHParams: string[];
  selectedMaxTrial: number;
  selectedMetric: MetricName;
  selectedView: GridListView;
  type: ExperimentVisualizationType,
}

const TOP_TRIALS_OPTIONS = [ 1, 10, 20, 50, 100 ];
const BATCH_MARGIN_OPTIONS = [ 1, 5, 10, 20, 50 ];

const ExperimentVisualizationFilters: React.FC<Props> = ({
  batches,
  hParams,
  metrics,
  onBatchChange,
  onBatchMarginChange,
  onHParamChange,
  onMaxTrialsChange,
  onMetricChange,
  onViewChange,
  selectedBatch,
  selectedBatchMargin,
  selectedHParams,
  selectedMaxTrial,
  selectedMetric,
  selectedView,
  type,
}: Props) => {
  const [ showMaxTrials, showBatches, showMetrics, showHParams, showLayout ] = useMemo(() => {
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

  const handleMaxTrialsChange = useCallback((count: SelectValue) => {
    if (onMaxTrialsChange) onMaxTrialsChange(count as number);
  }, [ onMaxTrialsChange ]);

  const handleBatchChange = useCallback((batch: SelectValue) => {
    if (onBatchChange) onBatchChange(batch as number);
  }, [ onBatchChange ]);

  const handleBatchMarginChange = useCallback((margin: SelectValue) => {
    if (onBatchMarginChange) onBatchMarginChange(margin as number);
  }, [ onBatchMarginChange ]);

  const handleHParamChange = useCallback((hps: SelectValue) => {
    if (!onHParamChange) return;
    onHParamChange(Array.isArray(hps) && hps.length !== 0 ? hps as string[] : undefined);
  }, [ onHParamChange ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    if (onMetricChange) onMetricChange(metric);
  }, [ onMetricChange ]);

  const handleViewChange = useCallback((view: GridListView) => {
    if (onViewChange) onViewChange(view);
  }, [ onViewChange ]);

  return (
    <div className={css.base}>
      {showMaxTrials && (
        <SelectFilter
          enableSearchFilter={false}
          label="Top Trials"
          showSearch={false}
          style={{ width: 70 }}
          value={selectedMaxTrial}
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
            value={selectedBatch}
            onChange={handleBatchChange}>
            {batches.map(batch => <Option key={batch} value={batch}>{batch}</Option>)}
          </SelectFilter>
          <SelectFilter
            enableSearchFilter={false}
            label="Batch Margin"
            showSearch={false}
            value={selectedBatchMargin}
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
          value={selectedMetric}
          width={'100%'}
          onChange={handleMetricChange} />
      )}
      {showHParams && (
        <MultiSelect
          label="HP"
          value={selectedHParams}
          onChange={handleHParamChange}>
          {hParams.map(hParam => <Option key={hParam} value={hParam}>{hParam}</Option>)}
        </MultiSelect>
      )}
      {showLayout && <GridListRadioGroup value={selectedView} onChange={handleViewChange} />}
    </div>
  );
};

export default ExperimentVisualizationFilters;
