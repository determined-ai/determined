import { Col, Row, Select } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';

import Link from 'components/Link';
import MetricSelectFilter from 'components/MetricSelectFilter';
import SelectFilter from 'components/SelectFilter';
import { V1MetricBatchesResponse, V1MetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentDetails, MetricName, MetricType } from 'types';

import css from './ExperimentVisualization.module.scss';

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
  const [ searchMetric, setSearchMetric ] = useState<string>();
  const [ batches, setBatches ] = useState<number[]>([]);
  const [ selectedBatch, setSelectedBatch ] = useState<number>();
  const [ canceler ] = useState(new AbortController());

  const metrics = [
    ...(validationMetrics || []).map(name => ({ name, type: MetricType.Validation })),
    ...(trainingMetrics || []).map(name => ({ name, type: MetricType.Training })),
  ];

  const handleMetricChange = useCallback((metric: MetricName) => {
    setSelectedMetric(metric);
  }, []);

  // Sets the default sub route
  useEffect(() => {
    if (type && (!TYPE_KEYS.includes(type) || type === DEFAULT_TYPE_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, type ]);

  // Stream available metrics
  useEffect(() => {
    consumeStream<V1MetricNamesResponse>(
      detApi.StreamingInternal.determinedMetricNames(experiment.id, { signal: canceler.signal }),
      event => {
        setSearchMetric(event.searcherMetric);
        setTrainingMetrics(event.trainingMetrics || []);
        setValidationMetrics(event.validationMetrics || []);
      },
    );

    return canceler.abort;
  }, [ canceler, experiment.id ]);

  // Stream available batches
  useEffect(() => {
    if (!selectedMetric) return;

    const batchesCanceler = new AbortController();
    const metricTypeParam = selectedMetric?.type === MetricType.Training
      ? 'METRIC_TYPE_TRAINING' : 'METRIC_TYPE_VALIDATION';
    consumeStream<V1MetricBatchesResponse>(
      detApi.StreamingInternal.determinedMetricBatches(
        experiment.id,
        selectedMetric?.name,
        metricTypeParam,
        { signal: batchesCanceler.signal },
      ),
      event => {
        if (event.batches) {
          setBatches(event.batches);
          if (event.batches.length !== 0) setSelectedBatch(event.batches[0]);
        }
      },
    );

    return () => batchesCanceler.abort();
  }, [ experiment.id, selectedMetric ]);

  // Set the default metric of interest
  useEffect(() => {
    if (selectedMetric) return;
    if (searchMetric) setSelectedMetric({ name: searchMetric, type: MetricType.Validation });
  }, [ searchMetric, selectedMetric ]);

  return (
    <div className={css.base}>
      <Row>
        <Col
          lg={{ order: 1, span: 20 }}
          md={{ order: 1, span: 18 }}
          sm={{ order: 2, span: 24 }}
          span={24}
          xs={{ order: 2, span: 24 }}>
          {experiment.id}
          {experiment.config.description}
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
            <div className={css.divider} />
            <header>Filters</header>
            <div className={css.filters}>
              <SelectFilter
                className={css.mobileNav}
                label="Chart Type"
                value={typeKey}
                verticalLayout={true}>
                {MENU.map(item => (
                  <Option key={item.type} value={item.type}>{item.label}</Option>
                ))}
              </SelectFilter>
              <MetricSelectFilter
                defaultMetricNames={metrics}
                metricNames={metrics}
                multiple={false}
                value={selectedMetric}
                verticalLayout={true}
                width={'100%'}
                onChange={handleMetricChange} />
              <SelectFilter
                label="Batches"
                style={{ width: '100%' }}
                value={selectedBatch}
                verticalLayout={true}>
                {batches.map(batch => <Option key={batch} value={batch}>{batch}</Option>)}
              </SelectFilter>
            </div>
          </div>
        </Col>
      </Row>
    </div>
  );
};

export default ExperimentVisualization;
