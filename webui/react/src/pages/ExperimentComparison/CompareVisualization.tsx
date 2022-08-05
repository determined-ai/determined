import { Tabs } from 'antd';
import queryString from 'query-string';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router';

import { useStore } from 'contexts/Store';
import { getExperimentDetails } from 'services/api';
import {
  V1ExpCompareMetricNamesResponse, V1ExpCompareTrialsSampleResponse,
} from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import { Primitive } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { flattenObject } from 'shared/utils/data';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentVisualizationType,
  Hyperparameter,
  HyperparameterType, MetricName, MetricType, metricTypeParamMap,
} from 'types';
import { Scale } from 'types';

import css from './CompareVisualization.module.scss';
import CompareCurve from './CompareVisualization/CompareCurve';
import CompareFilters, {
  ViewType, VisualizationFilters,
} from './CompareVisualization/CompareFilters';
import { TrialHParams } from './CompareVisualization/CompareTable';

enum PageError {
  MetricBatches,
  MetricHpImportance,
  MetricNames,
  ExperimentSample
}
export type HpValsMap = Record<string, Set<Primitive>>

const DEFAULT_TYPE_KEY = ExperimentVisualizationType.LearningCurve;
const DEFAULT_BATCH = 0;
const DEFAULT_BATCH_MARGIN = 10;
const DEFAULT_MAX_TRIALS = 100;
const DEFAULT_VIEW = ViewType.Grid;
const PAGE_ERROR_MESSAGES = {
  [PageError.MetricBatches]: 'Unable to retrieve experiment batches info.',
  [PageError.MetricHpImportance]: 'Unable to retrieve experiment hp importance.',
  [PageError.MetricNames]: 'Unable to retrieve experiment metric info.',
  [PageError.ExperimentSample]: 'Unable to retrieve experiment info.',
};
const CompareVisualization: React.FC = () => {

  const { ui } = useStore();

  const fullHParams = useRef<string[]>(
    [],
  );

  const defaultFilters: VisualizationFilters = {
    batch: DEFAULT_BATCH,
    batchMargin: DEFAULT_BATCH_MARGIN,
    hParams: [],
    maxTrial: DEFAULT_MAX_TRIALS,
    scale: Scale.Linear,
    view: DEFAULT_VIEW,
  };

  const location = useLocation();

  const experimentIds: number[] = useMemo(() => {
    const query = queryString.parse(location.search);
    if (query.id && typeof query.id === 'string'){
      return [ parseInt(query.id) ];
    } else if (Array.isArray(query.id)){
      return query.id.map((x) => parseInt(x));
    }
    return [];

  }, [ location.search ]);

  const [ filters, setFilters ] = useState<VisualizationFilters>(defaultFilters);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);

  const [ pageError, setPageError ] = useState<PageError>();

  useEffect(() => {
    if (filters.metric) return;
    const id = experimentIds[0];
    getExperimentDetails({ id }).then((experiment) => {
      const metric = { name: experiment.config.searcher.metric, type: MetricType.Validation };
      setFilters((filters) => ({ ...filters, metric }));

    });
  }, [ filters.metric, experimentIds ]);
  //
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);

  const [ hyperparameters, setHyperparameters ] = useState<Record<string, Hyperparameter>>({});
  const [ hpVals, setHpVals ] = useState<HpValsMap>({});
  const typeKey = DEFAULT_TYPE_KEY;
  const hasLoaded = useMemo(() => !!(trialIds.length && metrics.length > 0), [ metrics, trialIds ]);

  const handleFiltersChange = useCallback((filters: VisualizationFilters) => {
    setFilters(filters);
  }, [ ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    setFilters((filters) => ({ ...filters, metric }));
  }, []);

  useEffect(() => {
    if (ui.isPageHidden || !experimentIds.length || !filters.metric?.name) return;

    const canceler = new AbortController();
    const trialIdsMap: Record<number, number> = {};
    const trialDataMap: Record<number, number[]> = {};
    const trialHpMap: Record<number, TrialHParams> = {};
    const hpValsMap: HpValsMap = {};
    const batchesMap: Record<number, number> = {};
    const metricsMap: Record<number, Record<number, number>> = {};
    const hyperparameters: Record<string, Hyperparameter> = {};
    const metricTypeParam = metricTypeParamMap[filters.metric.type];

    readStream<V1ExpCompareTrialsSampleResponse>(
      detApi.StreamingInternal.expCompareTrialsSample(
        experimentIds,
        filters.metric.name,
        metricTypeParam,
        filters.maxTrial,
        undefined,
        undefined,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event || !event.trials) return;

        (event.promotedTrials || []).forEach((trialId) => trialIdsMap[trialId] = trialId);
        // (event.demotedTrials || []).forEach(trialId => delete trialIdsMap[trialId]);
        const newTrialIds = Object.values(trialIdsMap);
        setTrialIds((prevTrialIds) =>
          isEqual(prevTrialIds, newTrialIds)
            ? prevTrialIds
            : newTrialIds);

        (event.trials || []).forEach((trial) => {
          const id = trial.trialId;
          const flatHParams = flattenObject(trial.hparams || {});
          Object.keys(flatHParams).forEach(
            (hpParam) => {
              // distinguishing between constant vs not is irrelevant when constant
              // hps can vary across experiments. placeholder code
              (hyperparameters[hpParam] = { type: HyperparameterType.Constant });
              //
              if (hpValsMap[hpParam] == null) {
                hpValsMap[hpParam] = new Set([ flatHParams[hpParam] ]);
              } else {
                hpValsMap[hpParam].add(flatHParams[hpParam]);
              }
            },
          );
          setHyperparameters(hyperparameters);
          const hasHParams = Object.keys(flatHParams).length !== 0;

          if (hasHParams && !trialHpMap[id]) {
            trialHpMap[id] = {
              experimentId: trial.experimentId,
              hparams: flatHParams,
              id,
              metric: null,
            };
          }

          trialDataMap[id] = trialDataMap[id] || [];
          metricsMap[id] = metricsMap[id] || {};

          trial.data.forEach((datapoint) => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            trialHpMap[id].metric = datapoint.value;
          });
        });

        Object.keys(hpValsMap).forEach((hpParam) => {
          const hpVals = hpValsMap[hpParam];
          if (!hpVals.has('-') && newTrialIds.some((id) => trialHpMap[id] == null)) {
            hpValsMap[hpParam].add('-');
          }
        });
        setHpVals(hpValsMap);

        const newTrialHps = newTrialIds.map((id) => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

        const newChartData = newTrialIds.map((trialId) => newBatches.map((batch) => {
          const value = metricsMap[trialId][batch];
          return Number.isFinite(value) ? value : null;
        }));
        setChartData(newChartData);

      },
    ).catch((e) => {
      setPageError(e);
    });

    return () => canceler.abort();
  }, [ filters.metric, ui.isPageHidden, filters.maxTrial, experimentIds ]);

  useEffect(() => {
    if (ui.isPageHidden || !trialIds?.length) return;

    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    readStream<V1ExpCompareMetricNamesResponse>(
      detApi.StreamingInternal.expCompareMetricNames(
        trialIds,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event) return;
        (event.trainingMetrics || []).forEach((metric) => trainingMetricsMap[metric] = true);
        (event.validationMetrics || []).forEach((metric) => validationMetricsMap[metric] = true);

        const newTrainingMetrics = Object.keys(trainingMetricsMap).sort(alphaNumericSorter);
        const newValidationMetrics = Object.keys(validationMetricsMap).sort(alphaNumericSorter);
        const newMetrics = [
          ...(newValidationMetrics || []).map((name) => ({ name, type: MetricType.Validation })),
          ...(newTrainingMetrics || []).map((name) => ({ name, type: MetricType.Training })),
        ];
        setMetrics(newMetrics);
      },
    ).catch(() => {
      setPageError(PageError.MetricNames);
    });

    return () => canceler.abort();
  }, [ trialIds, ui.isPageHidden ]);

  if (!experimentIds.length) {
    return (
      <div className={css.alert}>
        <Spinner center className={css.alertSpinner} />
      </div>
    );
  } else if (pageError) {
    return <Message title={PAGE_ERROR_MESSAGES[pageError]} type={MessageType.Alert} />;
  }

  if (!metrics.length) {
    return <Spinner tip="Fetching metrics..." />;
  }

  const visualizationFilters = (
    <CompareFilters
      batches={batches || []}
      filters={filters}
      fullHParams={fullHParams.current}
      metrics={metrics || []}
      type={typeKey}
      onChange={handleFiltersChange}
      onMetricChange={handleMetricChange}
    />
  );
  return (
    <div className={css.base}>
      <Tabs
        activeKey={typeKey}
        destroyInactiveTabPane
        type="card">
        <Tabs.TabPane
          key={ExperimentVisualizationType.LearningCurve}
          tab="Learning Curve">
          {(experimentIds.length > 0 && filters.metric?.name && (
            <CompareCurve
              batches={batches}
              chartData={chartData}
              filters={visualizationFilters}
              hasLoaded={hasLoaded}
              hpVals={hpVals}
              hyperparameters={hyperparameters}
              selectedMaxTrial={filters.maxTrial}
              selectedMetric={filters.metric}
              selectedScale={filters.scale}
              trialHps={trialHps}
              trialIds={trialIds}
            />
          ))}
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
};

export default CompareVisualization;
