import { Select } from 'antd';
import { Col, Row } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';

import Link from 'components/Link';
import SelectFilter from 'components/SelectFilter';
import { V1MetricBatchesResponse, V1MetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ALL_VALUE, ExperimentDetails } from 'types';

import css from './ExperimentVisualization.module.scss';

const { OptGroup, Option } = Select;

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

  useEffect(() => {
    if (type && (!TYPE_KEYS.includes(type) || type === DEFAULT_TYPE_KEY)) {
      history.replace(basePath);
    }
  }, [ basePath, history, type ]);

  useEffect(() => {
    consumeStream<V1MetricNamesResponse>(
      detApi.StreamingInternal.determinedMetricNames(experiment.id),
      event => {
        console.log('event', event);
      },
    );
  }, [ experiment ]);

  return (
    <div className={css.base}>
      <Row>
        <Col md={20} span={24}>
          {experiment.id}
          {experiment.config.description}
        </Col>
        <Col md={4} span={24}>
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
            <div className={css.filters}>
              <header>Filters</header>
              <div>
                <SelectFilter label="Metric">
                  <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
                </SelectFilter>
              </div>
            </div>
          </div>
        </Col>
      </Row>
    </div>
  );
};

export default ExperimentVisualization;
