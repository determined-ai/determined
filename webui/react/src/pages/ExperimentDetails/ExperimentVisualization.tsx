import { type TabsProps } from 'antd';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import Message from 'components/kit/Message';
import Pivot from 'components/kit/Pivot';
import Spinner from 'components/kit/Spinner';
import useUI from 'components/kit/Theme';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import { paths } from 'routes/utils';
import { V1MetricBatchesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import store from 'stores/userSettings';
import {
  ExperimentBase,
  ExperimentSearcherName,
  HyperparameterType,
  Metric,
  MetricType,
  RunState,
  Scale,
  ValueOf,
} from 'types';
import { alphaNumericSorter } from 'utils/sort';

import ExperimentVisualizationFilters, {
  MAX_HPARAM_COUNT,
  ViewType,
  VisualizationFilters,
} from './ExperimentVisualization/ExperimentVisualizationFilters';
import HpHeatMaps from './ExperimentVisualization/HpHeatMaps';
import HpParallelCoordinates from './ExperimentVisualization/HpParallelCoordinates';
import HpScatterPlots from './ExperimentVisualization/HpScatterPlots';
import LearningCurve from './ExperimentVisualization/LearningCurve';
import css from './ExperimentVisualization.module.scss';

export const ExperimentVisualizationType = {
  HpHeatMap: 'hp-heat-map',
  HpParallelCoordinates: 'hp-parallel-coordinates',
  HpScatterPlots: 'hp-scatter-plots',
  LearningCurve: 'learning-curve',
} as const;

export type ExperimentVisualizationType = ValueOf<typeof ExperimentVisualizationType>;

interface Props {
  basePath: string;
  experiment: ExperimentBase;
  type?: ExperimentVisualizationType;
}

const PageError = {
  MetricBatches: 'MetricBatches',
  MetricNames: 'MetricNames',
} as const;

type PageError = ValueOf<typeof PageError>;

const STORAGE_PATH = 'experiment-visualization';
const STORAGE_FILTERS_KEY = 'filters';
const TYPE_KEYS = Object.values(ExperimentVisualizationType);
const DEFAULT_TYPE_KEY = ExperimentVisualizationType.LearningCurve;
const DEFAULT_BATCH = 0;
const DEFAULT_BATCH_MARGIN = 10;
const DEFAULT_MAX_TRIALS = 100;
const DEFAULT_VIEW = ViewType.Grid;
const PAGE_ERROR_MESSAGES = {
  [PageError.MetricBatches]: 'Unable to retrieve experiment batches info.',
  [PageError.MetricNames]: 'Unable to retrieve experiment metric info.',
};

const defaultFilters: VisualizationFilters = {
  batch: DEFAULT_BATCH,
  batchMargin: DEFAULT_BATCH_MARGIN,
  hParams: [],
  maxTrial: DEFAULT_MAX_TRIALS,
  metric: undefined,
  scale: Scale.Linear,
  view: DEFAULT_VIEW,
};

const ExperimentVisualization: React.FC<Props> = ({ basePath, experiment }: Props) => {
  const { ui } = useUI();
  const navigate = useNavigate();
  const location = useLocation();
  const searcherMetric = useRef<Metric>({
    group: MetricType.Validation,
    name: experiment.config.searcher.metric,
  });

  const { viz: type } = useParams<{ viz: ExperimentVisualizationType }>();
  const fullHParams = useRef<string[]>(
    Object.keys(experiment.hyperparameters || {}).filter((key) => {
      // Constant hyperparameters are not useful for visualizations.
      return experiment.hyperparameters[key].type !== HyperparameterType.Constant;
    }),
  );

  const storagePath = useMemo(
    () => `${STORAGE_PATH}/${experiment.id}/${STORAGE_FILTERS_KEY}`,
    [experiment],
  );
  const filtersLoadable = useObservable(store.get(VisualizationFilters, storagePath));

  const filters: VisualizationFilters = useMemo(() => {
    const filters = Loadable.getOrElse(defaultFilters, filtersLoadable);
    return filters || defaultFilters;
  }, [filtersLoadable]);

  const [typeKey, setTypeKey] = useState(() => {
    return type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  });
  const [batches, setBatches] = useState<number[]>();
  const [pageError, setPageError] = useState<PageError>();

  const handleMetricNamesError = useCallback(() => {
    setPageError(PageError.MetricNames);
  }, []);

  // Stream available metrics.
  const loadableMetrics = useMetricNames([experiment.id], handleMetricNamesError);
  const metrics = Loadable.getOrElse([], loadableMetrics);

  const { hasData, hasLoaded, isExperimentTerminal, isSupported } = useMemo(() => {
    return {
      hasData: !!metrics?.length,
      hasLoaded: Loadable.isLoaded(loadableMetrics) && !!batches,
      isExperimentTerminal: terminalRunStates.has(experiment.state),
      isSupported: !(
        ExperimentSearcherName.Single === experiment.config.searcher.name ||
        ExperimentSearcherName.Pbt === experiment.config.searcher.name
      ),
    };
  }, [batches, experiment, loadableMetrics, metrics]);

  const handleFiltersChange = useCallback(
    (newFilters: Partial<VisualizationFilters>) => {
      store.setPartial(VisualizationFilters, storagePath, newFilters);
    },
    [storagePath],
  );

  const getDefaultMetrics = useCallback(() => {
    const activeMetricFound = metrics.find(
      (metric) =>
        metric.group === searcherMetric.current.group &&
        metric.name === searcherMetric.current.name,
    );
    if (activeMetricFound) {
      return searcherMetric.current;
    } else if (metrics.length > 0) {
      return metrics[0];
    }
  }, [metrics]);

  const handleFiltersReset = useCallback(() => {
    store.set(VisualizationFilters, storagePath, {
      ...defaultFilters,
      batch: batches?.first() || DEFAULT_BATCH,
      hParams: fullHParams.current.slice(0, MAX_HPARAM_COUNT),
      metric: getDefaultMetrics(),
    });
  }, [storagePath, getDefaultMetrics, batches]);

  useEffect(() => {
    if (hasData && (!Loadable.isLoaded(filtersLoadable) || !filters.metric)) handleFiltersReset();
  }, [filtersLoadable, handleFiltersReset, filters.metric, hasData]);

  const handleTabChange = useCallback(
    (type: string) => {
      setTypeKey(type as ExperimentVisualizationType);
      navigate(`${basePath}/${type}`, { replace: true });
    },
    [basePath, navigate],
  );

  const visualizationFilters = useMemo(() => {
    return (
      <ExperimentVisualizationFilters
        batches={batches || []}
        filters={filters}
        fullHParams={fullHParams.current}
        metrics={metrics}
        type={typeKey}
        onChange={handleFiltersChange}
        onReset={handleFiltersReset}
      />
    );
  }, [batches, filters, handleFiltersChange, handleFiltersReset, metrics, typeKey]);

  const tabItems: TabsProps['items'] = useMemo(() => {
    /**
     * In the case of Custom Searchers, all the tabs besides
     * "Learning Curve" aren't helpful or relevant, so we are hiding them
     */
    if (
      filters.maxTrial === undefined ||
      filters.batchMargin === undefined ||
      filters.batch === undefined
    ) {
      return [];
    }

    const tabs: TabsProps['items'] = [
      {
        children: (
          <LearningCurve
            experiment={experiment}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedMaxTrial={filters.maxTrial}
            selectedMetric={filters.metric}
            selectedScale={filters.scale}
          />
        ),
        key: ExperimentVisualizationType.LearningCurve,
        label: 'Learning Curve',
      },
    ];
    if (experiment.config.searcher.name !== ExperimentSearcherName.Custom) {
      tabs.push(
        {
          children: (
            <HpParallelCoordinates
              experiment={experiment}
              filters={visualizationFilters}
              fullHParams={fullHParams.current}
              selectedBatch={filters.batch}
              selectedBatchMargin={filters.batchMargin}
              selectedHParams={filters.hParams}
              selectedMetric={filters.metric}
              selectedScale={filters.scale}
            />
          ),
          key: ExperimentVisualizationType.HpParallelCoordinates,
          label: 'HP Parallel Coordinates',
        },
        {
          children: (
            <HpScatterPlots
              experiment={experiment}
              filters={visualizationFilters}
              fullHParams={fullHParams.current}
              selectedBatch={filters.batch}
              selectedBatchMargin={filters.batchMargin}
              selectedHParams={filters.hParams}
              selectedMetric={filters.metric}
              selectedScale={filters.scale}
            />
          ),
          key: ExperimentVisualizationType.HpScatterPlots,
          label: 'HP Scatter Plots',
        },
        {
          children: (
            <HpHeatMaps
              experiment={experiment}
              filters={visualizationFilters}
              fullHParams={fullHParams.current}
              selectedBatch={filters.batch}
              selectedBatchMargin={filters.batchMargin}
              selectedHParams={filters.hParams}
              selectedMetric={filters.metric}
              selectedScale={filters.scale}
              selectedView={filters.view}
            />
          ),
          key: ExperimentVisualizationType.HpHeatMap,
          label: 'HP Heat Map',
        },
      );
    }
    return tabs;
  }, [
    experiment,
    filters.batch,
    filters.batchMargin,
    filters.hParams,
    filters.maxTrial,
    filters.metric,
    filters.scale,
    filters.view,
    visualizationFilters,
  ]);

  // Sets the default sub route.
  useEffect(() => {
    const isVisualizationRoute = location.pathname.includes(basePath);
    const isInvalidType = type && !TYPE_KEYS.includes(type);
    if (isVisualizationRoute && (!type || isInvalidType)) {
      navigate(`${basePath}/${typeKey}`, { replace: true });
    }
  }, [basePath, navigate, location, type, typeKey]);

  // Stream available batches.
  useEffect(() => {
    if (!isSupported || ui.isPageHidden || !filters.metric) return;

    const canceler = new AbortController();
    const metricTypeParam =
      filters.metric.group === MetricType.Training
        ? 'METRIC_TYPE_TRAINING'
        : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    readStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.metricBatches(
        experiment.id,
        filters.metric.name,
        metricTypeParam,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event) return;
        (event.batches || []).forEach((batch) => (batchesMap[batch] = batch));
        const newBatches = Object.values(batchesMap).sort(alphaNumericSorter);
        setBatches(newBatches);
      },
      () => setPageError(PageError.MetricBatches),
    );

    return () => canceler.abort();
  }, [filters.metric, experiment.id, filters.batch, isSupported, ui.isPageHidden]);

  if (!isSupported) {
    const alertMessage = `
      Hyperparameter visualizations are not applicable for single trial or PBT experiments.
    `;
    return (
      <div className={css.alert}>
        <Message
          description={
            <>
              Learn about&nbsp;
              <Link
                external
                path={paths.docs('/training-apis/experiment-config.html#searcher')}
                popout>
                how to run a hyperparameter search
              </Link>
              .
            </>
          }
          icon="warning"
          title={alertMessage}
        />
      </div>
    );
  } else if (experiment.state === RunState.Error) {
    return <Message icon="warning" title="No data to plot." />;
  } else if (pageError !== undefined) {
    return <Message icon="warning" title={PAGE_ERROR_MESSAGES[pageError]} />;
  } else if (!hasLoaded && experiment.state !== RunState.Paused) {
    return <Spinner spinning tip="Fetching metrics..." />;
  } else if (hasLoaded && !hasData) {
    return isExperimentTerminal || experiment.state === RunState.Paused ? (
      <Message icon="warning" title="No data to plot." />
    ) : (
      <div className={css.alert}>
        <Message
          description="Please wait until the experiment is further along."
          title="Not enough data points to plot."
        />
        <Spinner center spinning />
      </div>
    );
  }

  return (
    <div className={css.base}>
      <Pivot
        activeKey={typeKey}
        destroyInactiveTabPane
        items={tabItems}
        type="secondary"
        onChange={handleTabChange}
      />
    </div>
  );
};

export default ExperimentVisualization;
