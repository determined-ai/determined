import { Alert, Col, Row, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory } from 'react-router-dom';

import { GridListView } from 'components/GridListRadioGroup';
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
import ExperimentVisualizationFilters
  from './ExperimentVisualization/ExperimentVisualizationFilters';
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
const STORAGE_BATCH_KEY = 'batch';
const STORAGE_BATCH_MARGIN_KEY = 'batch-margin';
const STORAGE_HPARAMS_KEY = 'hyperparameters';
const STORAGE_MAX_TRIALS_KEY = 'max-trials';
const STORAGE_METRIC_KEY = 'metric';
const STORAGE_VIEW_KEY = 'grid-list-view';
const TYPE_KEYS = Object.values(ExperimentVisualizationType);
const DEFAULT_TYPE_KEY = ExperimentVisualizationType.LearningCurve;
const DEFAULT_BATCH_MARGIN = 10;
const DEFAULT_MAX_TRIALS = 100;
const MAX_HPARAM_COUNT = 10;
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
  const searcherMetric = { name: experiment.config.searcher.metric, type: MetricType.Validation };
  const defaultBatch = storage.get<number>(STORAGE_BATCH_KEY) || 0;
  const defaultBatchMargin = storage.get<number>(STORAGE_BATCH_MARGIN_KEY) || DEFAULT_BATCH_MARGIN;
  const defaultMaxTrial = storage.get<number>(STORAGE_MAX_TRIALS_KEY) || DEFAULT_MAX_TRIALS;
  const defaultMetric = storage.get<MetricName>(STORAGE_METRIC_KEY) || searcherMetric;
  const defaultView = storage.get<GridListView>(STORAGE_VIEW_KEY) || GridListView.Grid;
  const defaultTypeKey = type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  const [ typeKey, setTypeKey ] = useState(defaultTypeKey);
  const [ trainingMetrics, setTrainingMetrics ] = useState<string[]>([]);
  const [ validationMetrics, setValidationMetrics ] = useState<string[]>([]);
  const [ selectedMaxTrial, setSelectedMaxTrial ] = useState<number>(defaultMaxTrial);
  const [ selectedBatch, setSelectedBatch ] = useState<number>(defaultBatch);
  const [ selectedBatchMargin, setSelectedBatchMargin ] = useState<number>(defaultBatchMargin);
  const [ selectedMetric, setSelectedMetric ] = useState<MetricName>(defaultMetric);
  const [ selectedView, setSelectedView ] = useState(defaultView);
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<PageError>();

  const fullHParams = useMemo(() => {
    return (Object.keys(experiment.config.hyperparameters || {}).filter(key => {
      // Constant hyperparameters are not useful for visualizations
      const hp = experiment.config.hyperparameters[key];
      return hp.type !== ExperimentHyperParamType.Constant;
    }));
  }, [ experiment ]);
  const limitedHParams = useMemo(() => fullHParams.slice(0, MAX_HPARAM_COUNT), [ fullHParams ]);
  const defaultHParams = storage.get<string[]>(STORAGE_HPARAMS_KEY);
  const [
    selectedHParams,
    setSelectedHParams,
  ] = useState<string[]>(defaultHParams || limitedHParams);

  const metrics: MetricName[] = useMemo(() => ([
    ...(validationMetrics || []).map(name => ({ name, type: MetricType.Validation })),
    ...(trainingMetrics || []).map(name => ({ name, type: MetricType.Training })),
  ]), [ trainingMetrics, validationMetrics ]);

  const isExperimentTerminal = terminalRunStates.has(experiment.state);
  const hasBatches = batches.length !== 0;
  const hasMetrics = metrics.length !== 0;

  const handleBatchChange = useCallback((batch: number) => {
    storage.set(STORAGE_BATCH_KEY, batch);
    setSelectedBatch(batch);
  }, [ storage ]);

  const handleBatchMarginChange = useCallback((margin: number) => {
    storage.set(STORAGE_BATCH_MARGIN_KEY, margin);
    setSelectedBatchMargin(margin);
  }, [ storage ]);

  const handleHParamChange = useCallback((hParams?: string[]) => {
    if (!hParams) {
      storage.remove(STORAGE_HPARAMS_KEY);
      setSelectedHParams(limitedHParams);
    } else {
      storage.set(STORAGE_HPARAMS_KEY, hParams);
      setSelectedHParams(hParams);
    }
  }, [ limitedHParams, storage ]);

  const handleMaxTrialsChange = useCallback((count: number) => {
    storage.set(STORAGE_MAX_TRIALS_KEY, count);
    setSelectedMaxTrial(count);
  }, [ storage ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    storage.set(STORAGE_METRIC_KEY, metric);
    setSelectedMetric(metric);
  }, [ storage ]);

  const handleViewChange = useCallback((view: GridListView) => {
    storage.set(STORAGE_VIEW_KEY, view);
    setSelectedView(view);
  }, [ storage ]);

  const handleChartTypeChange = useCallback((type: SelectValue) => {
    setTypeKey(type as ExperimentVisualizationType);
    history.replace(type === DEFAULT_TYPE_KEY ? basePath : `${basePath}/${type}`);
  }, [ basePath, history ]);

  // Sets the default sub route
  useEffect(() => {
    if (type && (!TYPE_KEYS.includes(type) || type === DEFAULT_TYPE_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, type ]);

  // Stream available metrics
  useEffect(() => {
    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    setHasLoaded(false);

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
        setTrainingMetrics(newTrainingMetrics);
        setValidationMetrics(newValidationMetrics);
      },
    ).catch(() => {
      setHasLoaded(true);
      setPageError(PageError.MetricNames);
    });

    return () => canceler.abort();
  }, [ experiment.id ]);

  // Stream available batches
  useEffect(() => {
    if (!selectedMetric) return;

    const canceler = new AbortController();
    const metricTypeParam = selectedMetric?.type === MetricType.Training
      ? 'METRIC_TYPE_TRAINING' : 'METRIC_TYPE_VALIDATION';
    const batchesMap: Record<number, number> = {};

    consumeStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.determinedMetricBatches(
        experiment.id,
        selectedMetric?.name,
        metricTypeParam,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        if (!event) return;
        (event.batches || []).forEach(batch => batchesMap[batch] = batch);
        const newBatches = Object.values(batchesMap).sort(alphanumericSorter);
        setBatches(newBatches);
        if (newBatches.length !== 0 && !newBatches.includes(selectedBatch)) {
          setSelectedBatch(newBatches.first());
        }
        setHasLoaded(true);
      },
    ).catch(() => {
      setHasLoaded(true);
      setPageError(PageError.MetricBatches);
    });

    return () => canceler.abort();
  }, [ experiment.id, selectedBatch, selectedMetric ]);

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
  } else if (!hasMetrics || !hasBatches) {
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

  const filters = (
    <ExperimentVisualizationFilters
      batches={batches}
      hParams={fullHParams}
      metrics={metrics}
      selectedBatch={selectedBatch}
      selectedBatchMargin={selectedBatchMargin}
      selectedHParams={selectedHParams}
      selectedMaxTrial={selectedMaxTrial}
      selectedMetric={selectedMetric}
      selectedView={selectedView}
      type={typeKey}
      onBatchChange={handleBatchChange}
      onBatchMarginChange={handleBatchMarginChange}
      onHParamChange={handleHParamChange}
      onMaxTrialsChange={handleMaxTrialsChange}
      onMetricChange={handleMetricChange}
      onViewChange={handleViewChange}
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
              filters={filters}
              hParams={fullHParams}
              selectedMaxTrial={selectedMaxTrial}
              selectedMetric={selectedMetric}
            />
          )}
          {typeKey === ExperimentVisualizationType.HpParallelCoordinates && (
            <HpParallelCoordinates
              experiment={experiment}
              filters={filters}
              hParams={fullHParams}
              selectedBatch={selectedBatch}
              selectedBatchMargin={selectedBatchMargin}
              selectedHParams={selectedHParams}
              selectedMetric={selectedMetric}
            />
          )}
          {typeKey === ExperimentVisualizationType.HpScatterPlots && (
            <HpScatterPlots
              experiment={experiment}
              filters={filters}
              hParams={fullHParams}
              selectedBatch={selectedBatch}
              selectedBatchMargin={selectedBatchMargin}
              selectedHParams={selectedHParams}
              selectedMetric={selectedMetric}
            />
          )}
          {typeKey === ExperimentVisualizationType.HpHeatMap && (
            <HpHeatMaps
              experiment={experiment}
              filters={filters}
              hParams={fullHParams}
              selectedBatch={selectedBatch}
              selectedBatchMargin={selectedBatchMargin}
              selectedHParams={selectedHParams}
              selectedMetric={selectedMetric}
              selectedView={selectedView}
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
