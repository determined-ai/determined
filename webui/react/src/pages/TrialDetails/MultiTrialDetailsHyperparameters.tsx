import { Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import useMetricNames from 'hooks/useMetricNames';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import HpParallelCoordinates from 'pages/ExperimentDetails/ExperimentVisualization/HpParallelCoordinates';
import { V1MetricBatchesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import {
  ExperimentBase,
  ExperimentSearcherName,
  HyperparameterType,
  MetricType,
  TrialDetails,
} from 'types';
import handleError from 'utils/error';
import { alphaNumericSorter } from 'utils/sort';

import {
  ExperimentHyperparametersSettings,
  settingsConfigForExperimentHyperparameters,
} from './MultiTrialDetailsHyperparameters.settings';
import TrialDetailsHyperparameters from './TrialDetailsHyperparameters';

export interface Props {
  experiment: ExperimentBase;
  pageRef: React.RefObject<HTMLElement>; // TODO: This can be removed if TrialDetailsHyperparameters is refactored to use Glide
  trial: TrialDetails;
}

const MultiTrialDetailsHyperparameters: React.FC<Props> = ({
  experiment,
  pageRef,
  trial,
}: Props) => {
  const fullHParams = useRef<string[]>(
    Object.keys(experiment.hyperparameters || {}).filter((key) => {
      // Constant hyperparameters are not useful for visualizations.
      return experiment.hyperparameters[key].type !== HyperparameterType.Constant;
    }),
  );

  const settingsConfig = useMemo(
    () => settingsConfigForExperimentHyperparameters(experiment.id, trial.id, fullHParams.current),
    [experiment.id, trial.id],
  );

  const { settings, updateSettings, resetSettings, activeSettings } =
    useSettings<ExperimentHyperparametersSettings>(settingsConfig);

  const [batches, setBatches] = useState<number[]>();

  const filters: VisualizationFilters = useMemo(
    () => ({
      batch: settings.batch,
      batchMargin: settings.batchMargin,
      hParams: settings.hParams,
      maxTrial: settings.maxTrial,
      metric: settings.metric,
      scale: settings.scale,
    }),
    [
      settings.batch,
      settings.batchMargin,
      settings.hParams,
      settings.maxTrial,
      settings.metric,
      settings.scale,
    ],
  );

  // Stream available metrics.
  const metrics = useMetricNames([experiment.id], handleError);

  const isSupported = useMemo(() => {
    return !(
      ExperimentSearcherName.Single === experiment.config.searcher.name ||
      ExperimentSearcherName.Pbt === experiment.config.searcher.name
    );
  }, [experiment.config.searcher.name]);

  // Stream available batches.
  useEffect(() => {
    if (!isSupported || !settings.metric) return;

    const canceler = new AbortController();
    const metricTypeParam =
      settings.metric.type === MetricType.Training
        ? 'METRIC_TYPE_TRAINING'
        : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    readStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.metricBatches(
        experiment.id,
        settings.metric.name,
        metricTypeParam,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event) return;
        event.batches?.forEach((batch) => (batchesMap[batch] = batch));
        const newBatches = Object.values(batchesMap).sort(alphaNumericSorter);
        setBatches(newBatches);
      },
      handleError,
    );

    return () => canceler.abort();
  }, [experiment.id, isSupported, settings.metric]);

  // Set the default filter batch.
  useEffect(() => {
    if (!batches || batches.length === 0 || activeSettings(['batch']).includes('batch')) return;
    const bestValidationBatch = trial.bestValidationMetric?.totalBatches;
    updateSettings({
      batch:
        bestValidationBatch && batches.includes(bestValidationBatch)
          ? bestValidationBatch
          : batches.first(),
    });
  }, [
    activeSettings,
    batches,
    settings.batch,
    trial.bestValidationMetric?.totalBatches,
    updateSettings,
  ]);

  const handleFiltersChange = useCallback(
    (filters: Partial<VisualizationFilters>) => {
      updateSettings(filters);
    },
    [updateSettings],
  );

  const handleFiltersReset = useCallback(() => {
    resetSettings();
  }, [resetSettings]);

  // Set a default metric of interest filter.
  useEffect(() => {
    if (settings.metric !== undefined) return;
    const activeMetricFound = metrics.find(
      (metric) =>
        metric.type === MetricType.Validation && metric.name === experiment.config.searcher.metric,
    );
    updateSettings({ metric: activeMetricFound ?? metrics.first() });
  }, [experiment.config.searcher.metric, metrics, settings.metric, updateSettings]);

  const visualizationFilters = useMemo(() => {
    return (
      <ExperimentVisualizationFilters
        batches={batches || []}
        filters={filters}
        fullHParams={fullHParams.current}
        metrics={metrics}
        type={ExperimentVisualizationType.HpParallelCoordinates}
        onChange={handleFiltersChange}
        onReset={handleFiltersReset}
      />
    );
  }, [batches, handleFiltersChange, handleFiltersReset, metrics, filters]);

  return (
    <Space direction="vertical">
      <HpParallelCoordinates
        experiment={experiment}
        filters={visualizationFilters}
        focusedTrial={trial}
        fullHParams={fullHParams.current}
        selectedBatch={settings.batch}
        selectedBatchMargin={settings.batchMargin}
        selectedHParams={settings.hParams}
        selectedMetric={settings.metric}
        selectedScale={settings.scale}
      />
      <TrialDetailsHyperparameters pageRef={pageRef} trial={trial} />
    </Space>
  );
};

export default MultiTrialDetailsHyperparameters;
