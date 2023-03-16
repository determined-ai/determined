import { Alert } from 'antd';
import type { TabsProps } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import Link from 'components/Link';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import useStorage from 'hooks/useStorage';
import { paths } from 'routes/utils';
import {
  GetHPImportanceResponseMetricHPImportance,
  V1GetHPImportanceResponse,
  V1MetricBatchesResponse,
} from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner/Spinner';
import useUI from 'shared/contexts/stores/UI';
import { ValueOf } from 'shared/types';
import { hasObjectKeys } from 'shared/utils/data';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentBase,
  ExperimentSearcherName,
  HpImportanceMap,
  HpImportanceMetricMap,
  HyperparameterType,
  Metric,
  MetricType,
  RunState,
  Scale,
} from 'types';

import { hpImportanceSorter } from '../../utils/experiment';

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
  MetricHpImportance: 'MetricHpImportance',
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
  [PageError.MetricHpImportance]: 'Unable to retrieve experiment hp importance.',
  [PageError.MetricNames]: 'Unable to retrieve experiment metric info.',
};

const getHpImportanceMap = (hpImportanceMetrics: {
  [key: string]: GetHPImportanceResponseMetricHPImportance;
}): HpImportanceMetricMap => {
  const map: HpImportanceMetricMap = {};

  Object.keys(hpImportanceMetrics).forEach((metricName) => {
    map[metricName] = hpImportanceMetrics[metricName].hpImportance || {};
  });

  return map;
};

const ExperimentVisualization: React.FC<Props> = ({ basePath, experiment }: Props) => {
  const { ui } = useUI();
  const navigate = useNavigate();
  const location = useLocation();
  const storage = useStorage(`${STORAGE_PATH}/${experiment.id}`);
  const searcherMetric = useRef<Metric>({
    name: experiment.config.searcher.metric,
    type: MetricType.Validation,
  });

  const { viz: type } = useParams<{ viz: ExperimentVisualizationType }>();
  const fullHParams = useRef<string[]>(
    Object.keys(experiment.hyperparameters || {}).filter((key) => {
      // Constant hyperparameters are not useful for visualizations.
      return experiment.hyperparameters[key].type !== HyperparameterType.Constant;
    }),
  );

  const defaultFilters: VisualizationFilters = {
    batch: DEFAULT_BATCH,
    batchMargin: DEFAULT_BATCH_MARGIN,
    hParams: [],
    maxTrial: DEFAULT_MAX_TRIALS,
    metric: null,
    scale: Scale.Linear,
    view: DEFAULT_VIEW,
  };
  const initFilters = storage.getWithDefault<VisualizationFilters>(
    STORAGE_FILTERS_KEY,
    defaultFilters,
  );
  const [typeKey, setTypeKey] = useState(() => {
    return type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  });
  const [filters, setFilters] = useState<VisualizationFilters>(initFilters);
  const [activeMetric, setActiveMetric] = useState<Metric | null>(initFilters.metric);
  const [hasSearcherMetric, setHasSearcherMetric] = useState<boolean>(false);
  const [batches, setBatches] = useState<number[]>();
  const [hpImportanceMap, setHpImportanceMap] = useState<HpImportanceMap>();
  const [pageError, setPageError] = useState<PageError>();

  const handleMetricNamesError = useCallback(() => {
    setPageError(PageError.MetricNames);
  }, []);

  // Stream available metrics.
  const metrics = useMetricNames(experiment.id, handleMetricNamesError);

  const { hasData, hasLoaded, isExperimentTerminal, isSupported } = useMemo(() => {
    return {
      hasData: !!metrics?.length,
      hasLoaded: batches && metrics,
      isExperimentTerminal: terminalRunStates.has(experiment.state),
      isSupported: !(
        ExperimentSearcherName.Single === experiment.config.searcher.name ||
        ExperimentSearcherName.Pbt === experiment.config.searcher.name
      ),
    };
  }, [batches, experiment, metrics]);

  const hpImportance = useMemo(() => {
    if (!hpImportanceMap || !activeMetric) return {};
    return hpImportanceMap[activeMetric.type][activeMetric.name] || {};
  }, [activeMetric, hpImportanceMap]);

  const handleFiltersChange = useCallback(
    (filters: VisualizationFilters) => {
      setFilters(filters);
      storage.set(STORAGE_FILTERS_KEY, filters);
    },
    [storage],
  );

  const handleFiltersReset = useCallback(() => {
    storage.remove(STORAGE_FILTERS_KEY);
  }, [storage]);

  const handleMetricChange = useCallback((metric: Metric) => {
    setActiveMetric(metric);
  }, []);

  useEffect(() => {
    if (!hasSearcherMetric) {
      const activeMetricFound = metrics.find(
        (metric) =>
          metric.type === searcherMetric.current.type &&
          metric.name === searcherMetric.current.name,
      );
      if (activeMetricFound) {
        setHasSearcherMetric(true);
        setActiveMetric(searcherMetric.current);
        handleFiltersChange({
          ...filters,
          metric: searcherMetric.current,
        });
      }
    }
  }, [hasSearcherMetric, setActiveMetric, handleFiltersChange, filters, metrics]);

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
        hpImportance={hpImportance}
        metrics={metrics}
        type={typeKey}
        onChange={handleFiltersChange}
        onMetricChange={handleMetricChange}
        onReset={handleFiltersReset}
      />
    );
  }, [
    batches,
    filters,
    handleFiltersChange,
    handleFiltersReset,
    handleMetricChange,
    hpImportance,
    metrics,
    typeKey,
  ]);

  const tabItems: TabsProps['items'] = useMemo(() => {
    /**
     * In the case of Custom Searchers, all the tabs besides
     * "Learning Curve" aren't helpful or relevant, so we are hiding them
     */
    const tabs: TabsProps['items'] = [
      {
        children: (
          <LearningCurve
            experiment={experiment}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedMaxTrial={filters.maxTrial}
            selectedMetric={activeMetric}
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
              selectedMetric={activeMetric}
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
              selectedMetric={activeMetric}
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
              selectedMetric={activeMetric}
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
    activeMetric,
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

  useEffect(() => {
    if (!isSupported || ui.isPageHidden) return;
    const canceler = new AbortController();

    readStream<V1GetHPImportanceResponse>(
      detApi.StreamingInternal.getHPImportance(experiment.id, undefined, {
        signal: canceler.signal,
      }),
      (event) => {
        if (!event) return;
        setHpImportanceMap({
          [MetricType.Training]: getHpImportanceMap(event.trainingMetrics),
          [MetricType.Validation]: getHpImportanceMap(event.validationMetrics),
        });
      },
      () => setPageError(PageError.MetricHpImportance),
    );

    return () => canceler.abort();
  }, [experiment.id, filters?.metric, isSupported, metrics, ui.isPageHidden]);

  // Stream available batches.
  useEffect(() => {
    if (!isSupported || ui.isPageHidden || !activeMetric) return;

    const canceler = new AbortController();
    const metricTypeParam =
      activeMetric.type === MetricType.Training ? 'METRIC_TYPE_TRAINING' : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    readStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.metricBatches(
        experiment.id,
        activeMetric.name,
        metricTypeParam,
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
  }, [activeMetric, experiment.id, filters.batch, isSupported, ui.isPageHidden]);

  // Set the default filter batch.
  useEffect(() => {
    if (!batches || batches.length === 0) return;
    setFilters((prev) => {
      if (prev.batch !== DEFAULT_BATCH) return prev;
      return { ...prev, batch: batches.first() };
    });
  }, [batches]);

  // Update default filter hParams if not previously set.
  useEffect(() => {
    if (!isSupported) return;

    setFilters((prev) => {
      if (prev.hParams.length !== 0) return prev;
      const map = prev.metric ? hpImportanceMap?.[prev.metric.type]?.[prev.metric.name] || {} : {};
      let hParams = fullHParams.current;
      if (hasObjectKeys(map)) {
        hParams = hParams.sortAll((a, b) => hpImportanceSorter(a, b, map));
      }
      return { ...prev, hParams: hParams.slice(0, MAX_HPARAM_COUNT) };
    });
  }, [hpImportanceMap, isSupported]);

  if (!isSupported) {
    const alertMessage = `
      Hyperparameter visualizations are not applicable for single trial or PBT experiments.
    `;
    return (
      <div className={css.alert}>
        <Alert
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
          message={alertMessage}
          type="warning"
        />
      </div>
    );
  } else if (pageError !== undefined) {
    return <Message title={PAGE_ERROR_MESSAGES[pageError]} type={MessageType.Alert} />;
  } else if (!hasLoaded && experiment.state !== RunState.Paused) {
    return <Spinner tip="Fetching metrics..." />;
  } else if (!hasData) {
    return isExperimentTerminal || experiment.state === RunState.Paused ? (
      <Message title="No data to plot." type={MessageType.Empty} />
    ) : (
      <div className={css.alert}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot."
        />
        <Spinner center className={css.alertSpinner} />
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
