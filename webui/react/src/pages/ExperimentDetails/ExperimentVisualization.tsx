import { Alert, Tabs } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
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
  ExperimentVisualizationType, MetricName, MetricType,
} from 'types';
import { alphanumericSorter } from 'utils/sort';
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

type HpImportance = Record<string, number>;
type HpImportanceMap = Record<string, HpImportance>;

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
): HpImportanceMap => {
  const hpImportanceMap: HpImportanceMap = {};

  Object.keys(hpImportanceMetrics).forEach(metricName => {
    hpImportanceMap[metricName] = hpImportanceMetrics[metricName].hpImportance || {};
  });

  return hpImportanceMap;
};

const ExperimentVisualization: React.FC<Props> = ({
  basePath,
  experiment,
  type,
}: Props) => {
  const history = useHistory();
  const searcherMetric = useRef<MetricName>({
    name: experiment.config.searcher.metric,
    type: MetricType.Validation,
  });
  const storage = useStorage(`${STORAGE_PATH}/${experiment.id}`);
  const fullHParams = useRef<string[]>(
    (Object.keys(experiment.config.hyperparameters || {}).filter(key => {
      // Constant hyperparameters are not useful for visualizations.
      const hp = experiment.config.hyperparameters[key];
      return hp.type !== ExperimentHyperParamType.Constant;
    })),
  );
  const searcherMetric = useRef<MetricName>({
    name: experiment.config.searcher.metric,
    type: MetricType.Validation,
  });
  const [ typeKey, setTypeKey ] = useState(() => {
    return type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  });
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ metrics, setMetrics ] = useState<MetricName[]>([]);
  const [ filters, setFilters ] = useState<VisualizationFilters>(() => {
    const storedFilters = storage.get<VisualizationFilters>(STORAGE_FILTERS_KEY);
    return {
      batch: storedFilters?.batch || DEFAULT_BATCH,
      batchMargin: storedFilters?.batchMargin || DEFAULT_BATCH_MARGIN,
      hParams: storedFilters?.hParams || fullHParams.current.slice(0, MAX_HPARAM_COUNT),
      maxTrial: storedFilters?.maxTrial || DEFAULT_MAX_TRIALS,
      metric: storedFilters?.metric || searcherMetric.current,
      view: storedFilters?.view || DEFAULT_VIEW,
    };
  });
  const [ activeMetric, setActiveMetric ] = useState<MetricName>(filters.metric);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<PageError>();

  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const handleFiltersChange = useCallback((filters: VisualizationFilters) => {
    setFilters(filters);
    storage.set(STORAGE_FILTERS_KEY, filters);
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

        // Check to see if filter metric is valid.
        const filterMetricFound = newMetrics.reduce((acc, metric) => {
          return acc || (
            metric.type === filters.metric.type &&
            metric.name === filters.metric.name
          );
        }, false);
        if (!filterMetricFound) {
          setFilters(prev => ({ ...prev, metric: searcherMetric.current }));
          setActiveMetric(searcherMetric.current);
        }
      },
    ).catch(() => {
      setHasLoaded(true);
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
        const trainingHpRanks = getHpImportanceMap(event.trainingMetrics);
        const validationHpRanks = getHpImportanceMap(event.validationMetrics);
        setTrainingHpImportanceMap(trainingHpRanks);
        setValidationHpImportanceMap(validationHpRanks);
      },
    ).catch(() => {
      setHasLoaded(true);
      setPageError(PageError.MetricHpImportance);
    });

    return () => canceler.abort();
  }, [ experiment.id, filters.metric ]);

  // Stream available batches.
  useEffect(() => {
    const canceler = new AbortController();
    const metricTypeParam = activeMetric?.type === MetricType.Training
      ? 'METRIC_TYPE_TRAINING' : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    consumeStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.determinedMetricBatches(
        experiment.id,
        activeMetric?.name,
        metricTypeParam,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event) return;
        (event.batches || []).forEach(batch => batchesMap[batch] = batch);
        const newBatches = Object.values(batchesMap).sort(alphanumericSorter);
        setBatches(newBatches);
        if (filters.batch === 0) {
          setFilters(prev => ({ ...prev, batch: newBatches.first() }));
        }
        setHasLoaded(true);
      },
    ).catch(() => {
      setHasLoaded(true);
      setPageError(PageError.MetricBatches);
    });

    return () => canceler.abort();
  }, [ activeMetric, experiment.id, filters.batch ]);

  if (!hasLoaded) {
    return <Spinner />;
  } else if (pageError) {
    return <Message title={PAGE_ERROR_MESSAGES[pageError]} type={MessageType.Alert} />;
  } else if ([
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
  } else if (metrics.length === 0 || batches.length === 0) {
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
  }

  const visualizationFilters = (
    <ExperimentVisualizationFilters
      batches={batches}
      filters={filters}
      fullHParams={fullHParams.current}
      metrics={metrics}
      type={typeKey}
      onChange={handleFiltersChange}
      onMetricChange={handleMetricChange}
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
            hParams={fullHParams.current}
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
            hParams={fullHParams.current}
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
            hParams={fullHParams.current}
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
            hParams={fullHParams.current}
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
