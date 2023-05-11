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
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentBase,
  ExperimentSearcherName,
  HyperparameterType,
  Metric,
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
    () => settingsConfigForExperimentHyperparameters(experiment.id),
    [experiment.id],
  );

  const { settings, updateSettings, resetSettings } =
    useSettings<ExperimentHyperparametersSettings>(settingsConfig);

  const [batches, setBatches] = useState<number[]>();

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
    if (!isSupported || !settings.filters.metric) return;

    const canceler = new AbortController();
    const metricTypeParam =
      settings.filters.metric.type === MetricType.Training
        ? 'METRIC_TYPE_TRAINING'
        : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    readStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.metricBatches(
        experiment.id,
        settings.filters.metric.name,
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
  }, [experiment.id, isSupported, settings.filters.metric]);

  // Set the default filter batch.
  useEffect(() => {
    if (!batches || batches.length === 0 || settings.filters.batch !== 0) return;
    updateSettings({ filters: { ...settings.filters, batch: batches.first() } });
  }, [batches, settings.filters, updateSettings]);

  const handleFiltersChange = useCallback(
    (filters: VisualizationFilters) => {
      updateSettings({ filters });
    },
    [updateSettings],
  );

  const handleFiltersReset = useCallback(() => {
    resetSettings(['filters']);
  }, [resetSettings]);

  const handleMetricChange = useCallback(
    (metric: Metric) => {
      updateSettings({ filters: { ...settings.filters, metric } });
    },
    [settings.filters, updateSettings],
  );

  useEffect(() => {
    if (settings.filters.metric !== null) return;
    const activeMetricFound = metrics.find(
      (metric) =>
        metric.type === MetricType.Validation && metric.name === experiment.config.searcher.metric,
    );
    updateSettings({
      filters: { ...settings.filters, metric: activeMetricFound ?? metrics.first() },
    });
  }, [experiment.config.searcher.metric, metrics, settings.filters, updateSettings]);

  const visualizationFilters = useMemo(() => {
    return (
      <ExperimentVisualizationFilters
        batches={batches || []}
        filters={settings.filters}
        fullHParams={fullHParams.current}
        metrics={metrics}
        type={ExperimentVisualizationType.HpParallelCoordinates}
        onChange={handleFiltersChange}
        onMetricChange={handleMetricChange}
        onReset={handleFiltersReset}
      />
    );
  }, [
    batches,
    handleFiltersChange,
    handleFiltersReset,
    handleMetricChange,
    metrics,
    settings.filters,
  ]);

  return (
    <Space direction="vertical">
      <HpParallelCoordinates
        experiment={experiment}
        filters={visualizationFilters}
        fullHParams={fullHParams.current}
        selectedBatch={settings.filters.batch}
        selectedBatchMargin={settings.filters.batchMargin}
        selectedHParams={settings.filters.hParams}
        selectedMetric={settings.filters.metric}
        selectedScale={settings.filters.scale}
      />
      <TrialDetailsHyperparameters pageRef={pageRef} trial={trial} />
    </Space>
  );
};

export default MultiTrialDetailsHyperparameters;
