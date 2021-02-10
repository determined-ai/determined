import { Alert, Col, Row, Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
import { ExperimentBase, ExperimentSearcherName, MetricName, MetricType } from 'types';
import { alphanumericSorter } from 'utils/sort';
import { terminalRunStates } from 'utils/types';

import css from './ExperimentVisualization.module.scss';
import HpParallelCoordinates from './ExperimentVisualization/HpParallelCoordinates';
import HpScatterPlots from './ExperimentVisualization/HpScatterPlots';
import LearningCurve from './ExperimentVisualization/LearningCurve';

const { Option } = Select;

export enum VisualizationType {
  HpParallelCoordinates = 'hp-parallel-coordinates',
  HpHeatMap = 'hp-heat-map',
  HpScatterPlots = 'hp-scatter-plots',
  LearningCurve = 'learning-curve',
}

interface Props {
  basePath: string;
  experiment: ExperimentBase;
  type?: VisualizationType;
}

enum PageError {
  MetricBatches,
  MetricNames,
}

const STORAGE_PATH = 'experiment-visualization';
const TYPE_KEYS = Object.values(VisualizationType);
const DEFAULT_TYPE_KEY = VisualizationType.LearningCurve;
const MAX_HPARAM_COUNT = 20;
const MENU = [
  { label: 'Learning Curve', type: VisualizationType.LearningCurve },
  { label: 'HP Parallel Coordinates', type: VisualizationType.HpParallelCoordinates },
  { label: 'HP Scatter Plots', type: VisualizationType.HpScatterPlots },
  { disabled: true, label: 'HP Heat Map', type: VisualizationType.HpHeatMap },
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
  const storage = useStorage(STORAGE_PATH);
  const STORAGE_BATCH_KEY = `${experiment.id}/batch`;
  const STORAGE_METRIC_KEY = `${experiment.id}/metric`;
  const STORAGE_HPARAMS_KEY = `${experiment.id}/hyperparameters`;
  const defaultUserBatch = storage.get(STORAGE_BATCH_KEY) as number || 0;
  const defaultUserMetric = storage.get(STORAGE_METRIC_KEY) as MetricName || undefined;
  const defaultTypeKey = type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  const [ typeKey, setTypeKey ] = useState(defaultTypeKey);
  const [ trainingMetrics, setTrainingMetrics ] = useState<string[]>([]);
  const [ validationMetrics, setValidationMetrics ] = useState<string[]>([]);
  const [ selectedMetric, setSelectedMetric ] = useState<MetricName>(defaultUserMetric);
  const [ searcherMetric, setSearcherMetric ] = useState<string>();
  /* eslint-disable-next-line */
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ selectedBatch, setSelectedBatch ] = useState<number>(defaultUserBatch);

  const { fullHParams, limitedHParams } = useMemo(() => {
    // Constant hyperparameters are not useful for visualizations
    const fullHParams = (Object.keys(experiment.config.hyperparameters) || []).filter(key => {
      const hp = experiment.config.hyperparameters[key];
      return hp.type !== ExperimentHyperParamType.Constant;
    });
    const limitedHParams = fullHParams.slice(0, MAX_HPARAM_COUNT);
    return { fullHParams, limitedHParams };
  }, [ experiment ]);
  const defaultHParams = storage.get<string[]>(STORAGE_HPARAMS_KEY);
  const [ hParams, setHParams ] = useState<string[]>(defaultHParams || limitedHParams);

  const [ hasLoaded, setHasLoaded ] = useState(false);
  const [ pageError, setPageError ] = useState<PageError>();

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
  }, [ storage, STORAGE_BATCH_KEY ]);

  const handleHParamChange = useCallback((hParams?: string[]) => {
    if (!hParams) {
      storage.remove(STORAGE_HPARAMS_KEY);
      setHParams(limitedHParams);
    } else {
      storage.set(STORAGE_HPARAMS_KEY, hParams);
      setHParams(hParams);
    }
  }, [ limitedHParams, storage, STORAGE_HPARAMS_KEY ]);

  const handleMetricChange = useCallback((metric: MetricName) => {
    storage.set(STORAGE_METRIC_KEY, metric);
    setSelectedMetric(metric);
    setSelectedBatch(batches.first());
  }, [ batches, storage, STORAGE_METRIC_KEY ]);

  const handleChartTypeChange = useCallback((type: SelectValue) => {
    setTypeKey(type as VisualizationType);
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
        if (event.searcherMetric) setSearcherMetric(event.searcherMetric);
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
        setHasLoaded(true);

        if (!event) return;
        (event.batches || []).forEach(batch => batchesMap[batch] = batch);
        const newBatches = Object.values(batchesMap).sort(alphanumericSorter);
        setBatches(newBatches);
        if (selectedBatch === 0 && newBatches.length !== 0) {
          setSelectedBatch(newBatches.first());
        }
      },
    ).catch(() => {
      setHasLoaded(true);
      setPageError(PageError.MetricBatches);
    });

    return () => canceler.abort();
  }, [ experiment.id, selectedBatch, selectedMetric ]);

  // Set the default metric of interest
  useEffect(() => {
    if (selectedMetric) return;
    if (searcherMetric) setSelectedMetric({ name: searcherMetric, type: MetricType.Validation });
  }, [ metrics, searcherMetric, selectedMetric ]);

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

  return (
    <div className={css.base}>
      <Row>
        <Col
          lg={{ order: 1, span: 20 }}
          md={{ order: 1, span: 18 }}
          sm={{ order: 2, span: 24 }}
          span={24}
          xs={{ order: 2, span: 24 }}>
          {typeKey === VisualizationType.LearningCurve && (
            <LearningCurve
              experiment={experiment}
              hParams={fullHParams}
              metrics={metrics}
              selectedMetric={selectedMetric}
              onMetricChange={handleMetricChange}
            />
          )}
          {typeKey === VisualizationType.HpParallelCoordinates && (
            <HpParallelCoordinates
              batches={batches}
              experiment={experiment}
              hParams={fullHParams}
              metrics={metrics}
              selectedBatch={selectedBatch}
              selectedHParams={hParams}
              selectedMetric={selectedMetric}
              onBatchChange={handleBatchChange}
              onHParamChange={handleHParamChange}
              onMetricChange={handleMetricChange}
            />
          )}
          {typeKey === VisualizationType.HpScatterPlots && (
            <HpScatterPlots
              batches={batches}
              experiment={experiment}
              hParams={fullHParams}
              metrics={metrics}
              selectedBatch={selectedBatch}
              selectedHParams={hParams}
              selectedMetric={selectedMetric}
              onBatchChange={handleBatchChange}
              onHParamChange={handleHParamChange}
              onMetricChange={handleMetricChange}
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
                if (item.disabled) linkClasses.push(css.disabled);
                if (typeKey === item.type) linkClasses.push(css.active);

                const link = (
                  <Link
                    className={linkClasses.join(' ')}
                    disabled={item.disabled}
                    key={item.type}
                    path={`${basePath}/${item.type}`}
                    onClick={() => handleChartTypeChange(item.type)}>{item.label}</Link>
                );

                return item.disabled ? (
                  <Tooltip key={item.type} placement="left" title="Coming soon!">
                    <div>{link}</div>
                  </Tooltip>
                ) : link;
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
