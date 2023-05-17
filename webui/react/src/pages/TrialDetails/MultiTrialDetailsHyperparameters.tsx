import { Space } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import useMetricNames from 'hooks/useMetricNames';
import { useSettings } from 'hooks/useSettings';
import { ExperimentVisualizationType } from 'pages/ExperimentDetails/ExperimentVisualization';
import ExperimentVisualizationFilters, {
  ViewType,
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import HpParallelCoordinates from 'pages/ExperimentDetails/ExperimentVisualization/HpParallelCoordinates';
import { V1MetricBatchesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentBase,
  ExperimentSearcherName,
  HyperparameterType,
  MetricType,
  TrialDetails,
} from 'types';
import handleError from 'utils/error';

import {
  ExperimentHyperparametersSettings,
  settingsConfigForExperimentHyperparameters,
} from './MultiTrialDetailsHyperparameters.settings';
import TrialDetailsHyperparameters from './TrialDetailsHyperparameters';

export interface Props {
  experiment: ExperimentBase;
  pageRef: React.RefObject<HTMLElement>;
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
    () => settingsConfigForExperimentHyperparameters(experiment.id, fullHParams.current),
    [experiment.id],
  );

  const { settings, updateSettings, resetSettings } =
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
      view: ViewType.Grid, // View is required in the type but not used in parallel coordinates
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
  const metrics = useMetricNames(experiment.id, handleError);

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
    if (!batches || batches.length === 0 || settings.batch !== 0) return;
    updateSettings({ batch: batches.first() });
  }, [batches, settings.batch, updateSettings]);

  const handleFiltersChange = useCallback(
    (filters: VisualizationFilters) => {
      const { metric, batch, batchMargin, hParams, maxTrial, scale } = filters;
      updateSettings({ batch, batchMargin, hParams, maxTrial, metric, scale });
    },
    [updateSettings],
  );

  const handleFiltersReset = useCallback(() => {
    resetSettings(['filters']);
  }, [resetSettings]);

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
        fullHParams={fullHParams.current}
        selectedBatch={settings.batch}
        selectedBatchMargin={settings.batchMargin}
        selectedHParams={settings.hParams}
        selectedMetric={settings.metric}
        selectedScale={settings.scale}
        trial={trial}
      />
      <TrialDetailsHyperparameters pageRef={pageRef} trial={trial} />
    </Space>
  );
};

export default MultiTrialDetailsHyperparameters;
