import { Alert, Tabs } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

import Link from 'components/Link';
import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';
import {
  GetHPImportanceResponseMetricHPImportance,
  V1ExpTrial,
  V1GetHPImportanceResponse, V1MetricBatchesResponse, V1MetricNamesResponse, V1TrialsSampleResponse,
} from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import { hasObjectKeys, isEqual } from 'shared/utils/data';
// import { union } from 'shared/utils/set'
import { flattenObject } from 'shared/utils/data';
import {
  ExperimentBase, ExperimentSearcherName, ExperimentVisualizationType,
  HpImportanceMap, HpImportanceMetricMap, Hyperparameter, HyperparameterType, MetricName, MetricType, metricTypeParamMap,
} from 'types';
import { Scale } from 'types';
import { alphaNumericSorter, hpImportanceSorter } from 'utils/sort';

import css from './CompareVisualization.module.scss';
import LearningCurve from './CompareVisualization/CompareCurve';
import ExperimentVisualizationFilters, {
  MAX_HPARAM_COUNT, ViewType, VisualizationFilters,
} from './CompareVisualization/CompareFilters';
import { TrialHParams } from './CompareVisualization/CompareTable';

interface Props {
  basePath: string;
  experiments: ExperimentBase[];
  type?: ExperimentVisualizationType;
}

enum PageError {
  MetricBatches,
  MetricHpImportance,
  MetricNames,
  ExperimentSample
}

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

const getHpImportanceMap = (
  hpImportanceMetrics: { [key: string]: GetHPImportanceResponseMetricHPImportance },
): HpImportanceMetricMap => {
  const map: HpImportanceMetricMap = {};

  Object.keys(hpImportanceMetrics).forEach(metricName => {
    map[metricName] = hpImportanceMetrics[metricName].hpImportance || {};
  });

  return map;
};

const CompareVisualization: React.FC<Props> = ({
  basePath,
  experiments,
  type,
}: Props) => {
  const { ui } = useStore();
  //const storage = useStorage(`${STORAGE_PATH}/${experiment.id}`);

  const fullHParams = useRef<string[]>(
    (Object.keys(experiments?.[0]?.hyperparameters || {}).filter(key => {
      // Constant hyperparameters are not useful for visualizations.
      return experiments?.[0]?.hyperparameters?.[key]?.type !== HyperparameterType.Constant;
    })),
  );

  // const asd = useMemo(() => {
  //   const experimentHps = experiments.map(e => new Set(Object.keys(e.hyperparameters ?? {})))
  //   const allHParams = experimentHps.reduce(union)
  //   const differingHParams = [].filter(hp => {
  //     const allHpVals = experiments.map(e => e.hyperparameters[hp] )
  //   })

  // }, [])

  // Hack to show exp data
  const defaultMetric = {
    name: experiments?.[0]?.config.searcher.metric,
    type: MetricType.Validation,
  };

  const searcherMetric = useRef<MetricName>(defaultMetric);
  const defaultFilters: VisualizationFilters = {
    batch: DEFAULT_BATCH,
    batchMargin: DEFAULT_BATCH_MARGIN,
    hParams: [],
    maxTrial: DEFAULT_MAX_TRIALS,
    metric: searcherMetric.current,
    scale: Scale.Linear,
    view: DEFAULT_VIEW,
  };
  const initFilters = defaultFilters;

  const [ filters, setFilters ] = useState<VisualizationFilters>(initFilters);
  const [ activeMetric, setActiveMetric ] = useState<MetricName>(defaultMetric);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);
  const [ hpImportanceMap, setHpImportanceMap ] = useState<HpImportanceMap>();
  const [ pageError, setPageError ] = useState<PageError>();

  //
  const [ trialIds, setTrialIds ] = useState<number[]>([]);
  const [ chartData, setChartData ] = useState<(number | null)[][]>([]);
  const [ trialHps, setTrialHps ] = useState<TrialHParams[]>([]);
  const [ hyperparameters, setHyperparameters ] = useState<Record<string, Hyperparameter>>({});

  const typeKey = DEFAULT_TYPE_KEY;
  const { hasData, isSupported, hasLoaded } = useMemo(() => {
    return {
      hasData: (batches && batches.length !== 0) || (metrics && metrics.length !== 0) || (trialIds && trialIds.length > 0),
      hasLoaded: batches.length > 0 || metrics && metrics.length > 0 || trialIds.length > 0,
      // isSupported: ![
      //   ExperimentSearcherName.Single,
      //   ExperimentSearcherName.Pbt,
      // ].includes(experiments?.[0]?.config?.searcher?.name),
      isSupported: true,
    };
  }, [ batches, experiments, metrics ]);

  const hpImportance = useMemo(() => {
    if (!hpImportanceMap) return {};
    return hpImportanceMap[filters.metric.type][filters.metric.name] || {};
  }, [ filters.metric, hpImportanceMap ]);

  const handleFiltersChange = useCallback((filters: VisualizationFilters) => {
    setFilters(filters);
  }, [ ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    setActiveMetric(metric);
  }, []);

  useEffect(() => {
    if(!filters.metric?.name && experiments.length){
      setFilters({
        batch: DEFAULT_BATCH,
        batchMargin: DEFAULT_BATCH_MARGIN,
        hParams: [],
        maxTrial: DEFAULT_MAX_TRIALS,
        metric: {
          name: experiments[0].config.searcher.metric,
          type: MetricType.Validation,
        },
        scale: Scale.Linear,
        view: DEFAULT_VIEW,
      });
    }
  }, [ experiments.length ]);

  useEffect(() => {
    if (ui.isPageHidden || !experiments.length) return;

    const canceler = new AbortController();
    const trialIdsMap: Record<number, number> = {};
    const trialDataMap: Record<number, number[]> = {};
    const trialHpMap: Record<number, TrialHParams> = {};
    const batchesMap: Record<number, number> = {};
    const metricsMap: Record<number, Record<number, number>> = {};
    const hyperparameters: Record<string, Hyperparameter> = {};
    const metricTypeParam = metricTypeParamMap[filters.metric.type];

    readStream<V1TrialsSampleResponse>(
      detApi.StreamingInternal.experimentsSample(
        experiments.map(experiment => experiment.id),
        filters.metric.name,
        metricTypeParam,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event || !event.trials) return;

        /*
         * Cache trial ids, hparams, batches and metric values into easily searchable
         * dictionaries, then construct the necessary data structures to render the
         * chart and the table.
         */

        (event.promotedTrials || []).forEach(trialId => trialIdsMap[trialId] = trialId);
        (event.demotedTrials || []).forEach(trialId => delete trialIdsMap[trialId]);
        const newTrialIds = Object.values(trialIdsMap);
        setTrialIds(prevTrialIds =>
          isEqual(prevTrialIds, newTrialIds)
            ? prevTrialIds
            : newTrialIds);

        (event.trials || []).forEach(trial => {
          const id = trial.trialId;
          const flatHParams = flattenObject(trial.hparams || {});
          Object.keys(flatHParams).forEach(
            (hpParam) => (hyperparameters[hpParam] = { type: HyperparameterType.Constant }),
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

          trial.data.forEach(datapoint => {
            batchesMap[datapoint.batches] = datapoint.batches;
            metricsMap[id][datapoint.batches] = datapoint.value;
            trialHpMap[id].metric = datapoint.value;
          });
        });

        const newTrialHps = newTrialIds.map(id => trialHpMap[id]);
        setTrialHps(newTrialHps);

        const newBatches = Object.values(batchesMap);
        setBatches(newBatches);

        const newChartData = newTrialIds.map(trialId => newBatches.map(batch => {
          /**
           * TODO: filtering NaN, +/- Infinity for now, but handle it later with
           * dynamic min/max ranges via uPlot.Scales.
           */
          const value = metricsMap[trialId][batch];
          return Number.isFinite(value) ? value : null;
        }));
        setChartData(newChartData);

        // One successful event as come through.
      },
    ).catch(e => {
      setPageError(e);
    });

    return () => canceler.abort();
  }, [ trialIds, filters.metric, ui.isPageHidden ]);

  useEffect(() => {
    if (!isSupported || ui.isPageHidden || !trialIds?.length) return;

    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    readStream<V1MetricNamesResponse>(
      detApi.StreamingInternal.trialsMetricNames(
        trialIds,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event) return;
        /*
         * The metrics endpoint can intermittently send empty lists,
         * so we keep track of what we have seen on our end and
         * only add new metrics we have not seen to the list.
         */
        (event.trainingMetrics || []).forEach(metric => trainingMetricsMap[metric] = true);
        (event.validationMetrics || []).forEach(metric => validationMetricsMap[metric] = true);

        const newTrainingMetrics = Object.keys(trainingMetricsMap).sort(alphaNumericSorter);
        const newValidationMetrics = Object.keys(validationMetricsMap).sort(alphaNumericSorter);
        const newMetrics = [
          ...(newValidationMetrics || []).map(name => ({ name, type: MetricType.Validation })),
          ...(newTrainingMetrics || []).map(name => ({ name, type: MetricType.Training })),
        ];
        setMetrics(newMetrics);
      },
    ).catch(() => {
      setPageError(PageError.MetricNames);
    });

    return () => canceler.abort();
  }, [ trialIds, filters?.metric, isSupported, ui.isPageHidden ]);

  // Set the default filter batch.
  useEffect(() => {
    if (!batches || batches.length === 0) return;
    setFilters(prev => {
      if (prev.batch !== DEFAULT_BATCH) return prev;
      return { ...prev, batch: batches.first() };
    });
  }, [ batches ]);

  // Validate active metric against metrics.
  useEffect(() => {
    setActiveMetric(prev => {
      const activeMetricFound = (metrics || []).reduce((acc, metric) => {
        return acc || (metric.type === prev.type && metric.name === prev.name);
      }, false);
      return activeMetricFound ? prev : searcherMetric.current;
    });
  }, [ metrics ]);

  // Update default filter hParams if not previously set.
  useEffect(() => {
    if (!isSupported) return;

    setFilters(prev => {
      if (prev.hParams.length !== 0) return prev;
      const map = ((hpImportanceMap || {})[prev.metric.type] || {})[prev.metric.name];
      let hParams = fullHParams.current;
      if (hasObjectKeys(map)) {
        hParams = hParams.sortAll((a, b) => hpImportanceSorter(a, b, map));
      }
      return { ...prev, hParams: hParams.slice(0, MAX_HPARAM_COUNT) };
    });
  }, [ hpImportanceMap, isSupported ]);

  if (!experiments.length) {
    return (
      <div className={css.alert}>
        <Spinner center className={css.alertSpinner} />
      </div>
    );
  }if (!isSupported) {
    const alertMessage = `
      Hyperparameter visualizations are not applicable for single trial or PBT experiments.
    `;
    return (
      <div className={css.alert}>
        <Alert
          description={(
            <>
              Learn about&nbsp;
              <Link
                external
                path={paths.docs('/training-apis/experiment-config.html#searcher')}
                popout>how to run a hyperparameter search
              </Link>.
            </>
          )}
          message={alertMessage}
          type="warning"
        />
      </div>
    );
  } else if (pageError) {
    return <Message title={PAGE_ERROR_MESSAGES[pageError]} type={MessageType.Alert} />;
  } else if (!hasData && hasLoaded) {
    return (
      <div className={css.alert}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner center className={css.alertSpinner} />
      </div>
    );
  }

  const visualizationFilters = (
    <ExperimentVisualizationFilters
      batches={batches || []}
      filters={filters}
      fullHParams={fullHParams.current}
      hpImportance={hpImportance}
      metrics={metrics || []}
      type={typeKey}
      onChange={handleFiltersChange}
      onMetricChange={handleMetricChange}
      // onReset={handleFiltersReset}
    />
  );

  return (
    <div className={css.base}>
      <Tabs
        activeKey={typeKey}
        destroyInactiveTabPane
        type="card"
        onChange={() => {}}>
        <Tabs.TabPane
          key={ExperimentVisualizationType.LearningCurve}
          tab="Learning Curve">
          {(experiments.length > 0 && filters.metric?.name && (
            <LearningCurve
              batches={batches}
              chartData={chartData}
              experiments={experiments}
              filters={visualizationFilters}
              hasLoaded={hasLoaded}
              hyperparameters={hyperparameters}
              selectedMaxTrial={filters.maxTrial}
              selectedMetric={filters.metric}
              selectedScale={filters.scale}
              trialHps={trialHps}
              trialIds={trialIds}
            />
          ))}
        </Tabs.TabPane>
        {/* <Tabs.TabPane
          key={ExperimentVisualizationType.HpParallelCoordinates}
          tab="HP Parallel Coordinates">
          <HpParallelCoordinates
            experiments={experiments}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedBatch={filters.batch}
            selectedBatchMargin={filters.batchMargin}
            selectedHParams={filters.hParams}
            selectedMetric={filters.metric}
          />
        </Tabs.TabPane> */}
        {/* <Tabs.TabPane
          key={ExperimentVisualizationType.HpScatterPlots}
          tab="HP Scatter Plots">
          <HpScatterPlots
            experiments={experiments}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedBatch={filters.batch}
            selectedBatchMargin={filters.batchMargin}
            selectedHParams={filters.hParams}
            selectedMetric={filters.metric}
          />
        </Tabs.TabPane> */}
        {/* <Tabs.TabPane
          key={ExperimentVisualizationType.HpHeatMap}
          tab="HP Heat Map">
          <HpHeatMaps
            experiments={experiments}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedBatch={filters.batch}
            selectedBatchMargin={filters.batchMargin}
            selectedHParams={filters.hParams}
            selectedMetric={filters.metric}
            selectedView={filters.view}
          />
        </Tabs.TabPane> */}
      </Tabs>
    </div>
  );
};

export default CompareVisualization;
