import { Alert, Col, Row, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useHistory } from 'react-router-dom';

import Link from 'components/Link';
import Message, { MessageType } from 'components/Message';
import SelectFilter from 'components/SelectFilter';
import Spinner from 'components/Spinner';
import useStorage from 'hooks/useStorage';
import { paths } from 'routes/utils';
import { V1MetricBatchesResponse, V1MetricNamesResponse } from 'services/api-ts-sdk';
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
  MAX_HPARAM_COUNT, VisualizationFilters,
} from './ExperimentVisualization/ExperimentVisualizationFilters';
import HpHeatMaps from './ExperimentVisualization/HpHeatMaps';
import HpParallelCoordinates from './ExperimentVisualization/HpParallelCoordinates';
import HpScatterPlots from './ExperimentVisualization/HpScatterPlots';
import LearningCurve from './ExperimentVisualization/LearningCurve';

const { Option } = Select;

interface Props {
  basePath: string;
  experiment: ExperimentBase;
  type?: ExperimentVisualizationType;
}

enum PageError {
  MetricBatches,
  MetricNames,
}

const STORAGE_PATH = 'experiment-visualization';
const STORAGE_FILTERS_KEY = 'filters';
const TYPE_KEYS = Object.values(ExperimentVisualizationType);
const DEFAULT_TYPE_KEY = ExperimentVisualizationType.LearningCurve;
const DEFAULT_BATCH = 0;
const DEFAULT_BATCH_MARGIN = 10;
const DEFAULT_MAX_TRIALS = 100;
const MENU = [
  { label: 'Learning Curve', type: ExperimentVisualizationType.LearningCurve },
  { label: 'HP Parallel Coordinates', type: ExperimentVisualizationType.HpParallelCoordinates },
  { label: 'HP Scatter Plots', type: ExperimentVisualizationType.HpScatterPlots },
  { label: 'HP Heat Map', type: ExperimentVisualizationType.HpHeatMap },
];
const PAGE_ERROR_MESSAGES = {
  [PageError.MetricBatches]: 'Unable to retrieve experiment batches info.',
  [PageError.MetricNames]: 'Unable to retrieve experiment metric info.',
};

const ExperimentVisualization: React.FC<Props> = ({
  basePath,
  experiment,
  type,
}: Props) => {
  const history = useHistory();
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
    };
  });
  const [ activeMetric, setActiveMetric ] = useState<MetricName>(filters.metric);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<PageError>();

  const isExperimentTerminal = terminalRunStates.has(experiment.state);

  const handleFiltersChange = useCallback((filters: VisualizationFilters) => {
    setFilters(filters);
  }, []);

  const handleMetricChange = useCallback((metric: MetricName) => {
    setActiveMetric(metric);
  }, []);

  const handleChartTypeChange = useCallback((type: SelectValue) => {
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
      <Row>
        <Col
          lg={{ order: 1, span: 20 }}
          md={{ order: 1, span: 18 }}
          sm={{ order: 2, span: 24 }}
          span={24}
          xs={{ order: 2, span: 24 }}>
          {typeKey === ExperimentVisualizationType.LearningCurve && (
            <LearningCurve
              experiment={experiment}
              filters={visualizationFilters}
              hParams={fullHParams.current}
              selectedMaxTrial={filters.maxTrial}
              selectedMetric={filters.metric}
            />
          )}
          {typeKey === ExperimentVisualizationType.HpParallelCoordinates && (
            <HpParallelCoordinates
              experiment={experiment}
              filters={visualizationFilters}
              hParams={fullHParams.current}
              selectedBatch={filters.batch}
              selectedBatchMargin={filters.batchMargin}
              selectedHParams={filters.hParams}
              selectedMetric={filters.metric}
            />
          )}
          {typeKey === ExperimentVisualizationType.HpScatterPlots && (
            <HpScatterPlots
              experiment={experiment}
              filters={visualizationFilters}
              hParams={fullHParams.current}
              selectedBatch={filters.batch}
              selectedBatchMargin={filters.batchMargin}
              selectedHParams={filters.hParams}
              selectedMetric={filters.metric}
            />
          )}
          {typeKey === ExperimentVisualizationType.HpHeatMap && (
            <HpHeatMaps
              experiment={experiment}
              filters={visualizationFilters}
              hParams={fullHParams.current}
              selectedBatch={filters.batch}
              selectedBatchMargin={filters.batchMargin}
              selectedHParams={filters.hParams}
              selectedMetric={filters.metric}
            />
          )}
        </Col>
        <Col
          lg={{ order: 2, span: 4 }}
          md={{ order: 2, span: 6 }}
          sm={{ order: 1, span: 24 }}
          span={24}
          xs={{ order: 1, span: 24 }}>
          <div className={css.inspector}>
            <div className={css.menu}>
              {MENU.map(item => {
                const linkClasses = [ css.link ];
                if (typeKey === item.type) linkClasses.push(css.active);
                return (
                  <Link
                    className={linkClasses.join(' ')}
                    key={item.type}
                    path={`${basePath}/${item.type}`}
                    onClick={() => handleChartTypeChange(item.type)}>{item.label}</Link>
                );
              })}
            </div>
            <div className={css.mobileMenu}>
              <SelectFilter
                label="Chart Type"
                value={typeKey}
                onChange={handleChartTypeChange}>
                {MENU.map(item => (
                  <Option key={item.type} value={item.type}>{item.label}</Option>
                ))}
              </SelectFilter>
            </div>
          </div>
        </Col>
      </Row>
    </div>
  );
};

export default ExperimentVisualization;
