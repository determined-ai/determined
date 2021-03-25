import { Alert, Tabs } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useHistory } from 'react-router-dom';

import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import Spinner from 'components/Spinner';
import useStorage from 'hooks/useStorage';
import { paths } from 'routes/utils';
import {
  GetHPImportanceResponseMetricHPImportance,
  V1GetHPImportanceResponse, V1MetricBatchesResponse, V1MetricNamesResponse,
} from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import {
  ExperimentBase, ExperimentHyperParamType, ExperimentSearcherName,
  ExperimentVisualizationType, HpImportanceMap, HpImportanceMetricMap, MetricName, MetricType,
} from 'types';
import { hasObjectKeys } from 'utils/data';
import { alphanumericSorter, hpImportanceSorter } from 'utils/sort';
import { terminalRunStates } from 'utils/types';

import css from './ExperimentVisualization.module.scss';
import ExperimentVisualizationFilters, {
  MAX_HPARAM_COUNT, ViewType, VisualizationFilters,
} from './ExperimentVisualization/ExperimentVisualizationFilters';
import HpHeatMaps from './ExperimentVisualization/HpHeatMaps';
import HpParallelCoordinates from './ExperimentVisualization/HpParallelCoordinates';
import HpScatterPlots from './ExperimentVisualization/HpScatterPlots';
import LearningCurve from './ExperimentVisualization/LearningCurve';

interface Props {
  basePath: string;
  experiment: ExperimentBase;
  type?: ExperimentVisualizationType;
}

enum PageError {
  MetricBatches,
  MetricHpImportance,
  MetricNames,
}

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

const getHpImportanceMap = (
  hpImportanceMetrics: { [key: string]: GetHPImportanceResponseMetricHPImportance },
): HpImportanceMetricMap => {
  const map: HpImportanceMetricMap = {};

  Object.keys(hpImportanceMetrics).forEach(metricName => {
    map[metricName] = hpImportanceMetrics[metricName].hpImportance || {};
  });

  return map;
};

const ExperimentVisualization: React.FC<Props> = ({
  basePath,
  experiment,
  type,
}: Props) => {
  const history = useHistory();
  const storage = useStorage(`${STORAGE_PATH}/${experiment.id}`);
  const searcherMetric = useRef<MetricName>({
    name: experiment.config.searcher.metric,
    type: MetricType.Validation,
  });
  const fullHParams = useRef<string[]>(
    (Object.keys(experiment.config.hyperparameters || {}).filter(key => {
      // Constant hyperparameters are not useful for visualizations.
      const hp = experiment.config.hyperparameters[key];
      return hp.type !== ExperimentHyperParamType.Constant;
    })),
  );
  const defaultFilters: VisualizationFilters = {
    batch: DEFAULT_BATCH,
    batchMargin: DEFAULT_BATCH_MARGIN,
    hParams: [],
    maxTrial: DEFAULT_MAX_TRIALS,
    metric: searcherMetric.current,
    view: DEFAULT_VIEW,
  };
  const initFilters = storage.getWithDefault<VisualizationFilters>(
    STORAGE_FILTERS_KEY,
    defaultFilters,
  );
  const [ typeKey, setTypeKey ] = useState(() => {
    return type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  });
  const [ filters, setFilters ] = useState<VisualizationFilters>(initFilters);
  const [ activeMetric, setActiveMetric ] = useState<MetricName>(initFilters.metric);
  const [ batches, setBatches ] = useState<number[]>();
  const [ metrics, setMetrics ] = useState<MetricName[]>();
  const [ hpImportanceMap, setHpImportanceMap ] = useState<HpImportanceMap>();
  const [ pageError, setPageError ] = useState<PageError>();

  const { hasData, hasLoaded, isExperimentTerminal } = useMemo(() => {
    return {
      hasData: batches && batches.length !== 0 && metrics && metrics.length !== 0,
      hasLoaded: batches && metrics && hpImportanceMap,
      isExperimentTerminal: terminalRunStates.has(experiment.state),
    };
  }, [ batches, experiment.state, metrics, hpImportanceMap ]);

  const hpImportance = useMemo(() => {
    if (!hpImportanceMap) return {};
    return hpImportanceMap[filters.metric.type][filters.metric.name] || {};
  }, [ filters.metric, hpImportanceMap ]);

  const handleFiltersChange = useCallback((filters: VisualizationFilters) => {
    setFilters(filters);
    storage.set(STORAGE_FILTERS_KEY, filters);
  }, [ storage ]);

  const handleFiltersReset = useCallback(() => {
    storage.remove(STORAGE_FILTERS_KEY);
  }, [ storage ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    setActiveMetric(metric);
  }, []);

  const handleTabChange = useCallback((type: string) => {
    setTypeKey(type as ExperimentVisualizationType);
    history.replace(type === DEFAULT_TYPE_KEY ? basePath : `${basePath}/${type}`);
  }, [ basePath, history ]);

  // Sets the default sub route.
  useEffect(() => {
    if (type && (!TYPE_KEYS.includes(type) || type === DEFAULT_TYPE_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, type ]);

  // Stream available metrics.
  useEffect(() => {
    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    consumeStream<V1MetricNamesResponse>(
      detApi.StreamingInternal.determinedMetricNames(
        experiment.id,
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
        const newTrainingMetrics = Object.keys(trainingMetricsMap).sort(alphanumericSorter);
        const newValidationMetrics = Object.keys(validationMetricsMap).sort(alphanumericSorter);
        const newMetrics = [
          ...(newValidationMetrics || []).map(name => ({ name, type: MetricType.Validation })),
          ...(newTrainingMetrics || []).map(name => ({ name, type: MetricType.Training })),
        ];
        setMetrics(newMetrics);
      },
    ).catch(() => {
      setPageError(PageError.MetricNames);
    });

    consumeStream<V1GetHPImportanceResponse>(
      detApi.StreamingInternal.determinedGetHPImportance(
        experiment.id,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event) return;
        setHpImportanceMap({
          [MetricType.Training]: getHpImportanceMap(event.trainingMetrics),
          [MetricType.Validation]: getHpImportanceMap(event.validationMetrics),
        });
      },
    ).catch(() => {
      setPageError(PageError.MetricHpImportance);
    });

    return () => canceler.abort();
  }, [ experiment.id, filters?.metric ]);

  // Stream available batches.
  useEffect(() => {
    const canceler = new AbortController();
    const metricTypeParam = activeMetric.type === MetricType.Training
      ? 'METRIC_TYPE_TRAINING' : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    consumeStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.determinedMetricBatches(
        experiment.id,
        activeMetric.name,
        metricTypeParam,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event) return;
        (event.batches || []).forEach(batch => batchesMap[batch] = batch);
        const newBatches = Object.values(batchesMap).sort(alphanumericSorter);
        setBatches(newBatches);
      },
    ).catch(() => {
      setPageError(PageError.MetricBatches);
    });

    return () => canceler.abort();
  }, [ activeMetric, experiment.id, filters.batch ]);

  // Set the default filter batch.
  useEffect(() => {
    if (!batches || batches.length === 0) return;
    if (filters.batch !== DEFAULT_BATCH) return;
    setFilters(prev => ({ ...prev, batch: batches.first() }));
  }, [ batches, filters.batch ]);

  // Validate active metric against metrics.
  useEffect(() => {
    const activeMetricFound = (metrics || []).reduce((acc, metric) => {
      return acc || (metric.type === activeMetric.type && metric.name === activeMetric.name);
    }, false);
    if (!activeMetricFound) setActiveMetric(searcherMetric.current);
  }, [ activeMetric, metrics ]);

  // Update default filter hParams if not previously set.
  useEffect(() => {
    if (filters.hParams.length !== 0) return;

    setFilters(prev => {
      const map = ((hpImportanceMap || {})[filters.metric.type] || {})[filters.metric.name];
      let hParams = fullHParams.current;
      if (hasObjectKeys(map)) {
        hParams = hParams.sortAll((a, b) => hpImportanceSorter(a, b, map));
      }
      return { ...prev, hParams: hParams.slice(0, MAX_HPARAM_COUNT) };
    });
  }, [ filters, hpImportanceMap ]);

  if ([
    ExperimentSearcherName.Single,
    ExperimentSearcherName.Pbt,
  ].includes(experiment.config.searcher.name)) {
    const alertMessage = `
      Hyperparameter visualizations are not applicable for single trial or PBT experiments.
    `;
    return <Alert
      description={<>
      Learn about &nbsp;
        <Link
          external
          path={paths.docs('/reference/experiment-config.html#searcher')}
          popout>how to run a hyperparameter search</Link>.
      </>}
      message={alertMessage}
      type="warning" />;
  } else if (pageError) {
    return <Message title={PAGE_ERROR_MESSAGES[pageError]} type={MessageType.Alert} />;
  } else if (!hasData) {
    return isExperimentTerminal ? (
      <Message title="No data to plot." type={MessageType.Empty} />
    ) : (
      <div className={css.waiting}>
        <Alert
          description="Please wait until the experiment is further along."
          message="Not enough data points to plot." />
        <Spinner />
      </div>
    );
  } else if (!hasLoaded) {
    return <Spinner />;
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
      onReset={handleFiltersReset}
    />
  );

  return (
    <div className={css.base}>
      <Tabs activeKey={typeKey} className={css.base} type="card" onChange={handleTabChange}>
        <Tabs.TabPane
          key={ExperimentVisualizationType.LearningCurve}
          tab="Learning Curve">
          <LearningCurve
            experiment={experiment}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedMaxTrial={filters.maxTrial}
            selectedMetric={filters.metric}
          />
        </Tabs.TabPane>
        <Tabs.TabPane
          key={ExperimentVisualizationType.HpParallelCoordinates}
          tab="HP Parallel Coordinates">
          <HpParallelCoordinates
            experiment={experiment}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedBatch={filters.batch}
            selectedBatchMargin={filters.batchMargin}
            selectedHParams={filters.hParams}
            selectedMetric={filters.metric}
          />
        </Tabs.TabPane>
        <Tabs.TabPane
          key={ExperimentVisualizationType.HpScatterPlots}
          tab="HP Scatter Plots">
          <HpScatterPlots
            experiment={experiment}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedBatch={filters.batch}
            selectedBatchMargin={filters.batchMargin}
            selectedHParams={filters.hParams}
            selectedMetric={filters.metric}
          />
        </Tabs.TabPane>
        <Tabs.TabPane
          key={ExperimentVisualizationType.HpHeatMap}
          tab="HP Heat Map">
          <HpHeatMaps
            experiment={experiment}
            filters={visualizationFilters}
            fullHParams={fullHParams.current}
            selectedBatch={filters.batch}
            selectedBatchMargin={filters.batchMargin}
            selectedHParams={filters.hParams}
            selectedMetric={filters.metric}
            selectedView={filters.view}
          />
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
};

export default ExperimentVisualization;
