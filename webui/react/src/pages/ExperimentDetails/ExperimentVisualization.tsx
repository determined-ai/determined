import { Col, Row, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';

import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
import { V1MetricBatchesResponse, V1MetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentDetails, MetricName, MetricType } from 'types';
import { alphanumericSorter } from 'utils/data';

import css from './ExperimentVisualization.module.scss';
import LearningCurve from './ExperimentVisualization/LearningCurve';

const { Option } = Select;

export enum VisualizationType {
  HpParallelCoord = 'hp-parallel-coord',
  HpImportance = 'hp-importance',
  LearningCurve = 'learning-curve',
  ScatterPlots = 'scatter-plots',
}

interface Props {
  basePath: string;
  experiment: ExperimentDetails;
  type?: VisualizationType;
}

const TYPE_KEYS = Object.values(VisualizationType);
const DEFAULT_TYPE_KEY = VisualizationType.LearningCurve;
const MENU = [
  { label: 'Learning Curve', type: VisualizationType.LearningCurve },
  { label: 'HP Parallel Coordinates', type: VisualizationType.HpParallelCoord },
  { label: 'HP Importance', type: VisualizationType.HpImportance },
  { label: 'Scatter Plots', type: VisualizationType.ScatterPlots },
];

const ExperimentVisualization: React.FC<Props> = ({
  basePath,
  experiment,
  type,
}: Props) => {
  const history = useHistory();
  const defaultTypeKey = type && TYPE_KEYS.includes(type) ? type : DEFAULT_TYPE_KEY;
  const [ typeKey, setTypeKey ] = useState(defaultTypeKey);
  const [ trainingMetrics, setTrainingMetrics ] = useState<string[]>([]);
  const [ validationMetrics, setValidationMetrics ] = useState<string[]>([]);
  const [ selectedMetric, setSelectedMetric ] = useState<MetricName>();
  const [ searcherMetric, setSearcherMetric ] = useState<string>();
  /* eslint-disable-next-line */
  const [ batches, setBatches ] = useState<number[]>([]);

  const metrics: MetricName[] = [
    ...(validationMetrics || []).map(name => ({ name, type: MetricType.Validation })),
    ...(trainingMetrics || []).map(name => ({ name, type: MetricType.Training })),
  ];

  const handleMetricChange = useCallback((metric: MetricName) => {
    setSelectedMetric(metric);
  }, []);

  const handleChartTypeChange = useCallback((type: SelectValue) => {
    setTypeKey(type as VisualizationType);
  }, []);

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
    );

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
      },
    );

    return () => canceler.abort();
  }, [ experiment.id, selectedMetric ]);

  // Set the default metric of interest
  useEffect(() => {
    if (selectedMetric) return;
    if (searcherMetric) setSelectedMetric({ name: searcherMetric, type: MetricType.Validation });
  }, [ searcherMetric, selectedMetric ]);

  return (
    <div className={css.base}>
      <Row>
        <Col
          lg={{ order: 1, span: 20 }}
          md={{ order: 1, span: 18 }}
          sm={{ order: 2, span: 24 }}
          span={24}
          xs={{ order: 2, span: 24 }}>
          {selectedMetric && typeKey === VisualizationType.LearningCurve && (
            <LearningCurve
              experiment={experiment}
              metrics={metrics}
              selectedMetric={selectedMetric}
              onMetricChange={handleMetricChange} />
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
              {MENU.map(item => (
                <Link
                  className={typeKey === item.type ? css.active : undefined}
                  key={item.type}
                  path={`${basePath}/${item.type}`}
                  onClick={() => setTypeKey(item.type)}>{item.label}</Link>
              ))}
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
