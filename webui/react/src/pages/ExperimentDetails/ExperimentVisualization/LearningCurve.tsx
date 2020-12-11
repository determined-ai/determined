import React, { useEffect, useState } from 'react';

import LearningCurveChart from 'components/LearningCurveChart';
import Section from 'components/Section';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { ExperimentDetails, MetricName, metricTypeParamMap } from 'types';

import css from './LearningCurve.module.scss';

interface Props {
  experiment: ExperimentDetails;
  metric: MetricName;
}

const MAX_TRIALS = 100;
const MAX_DATAPOINTS = 5000;

const LearningCurve: React.FC<Props> = ({ experiment, metric }: Props) => {
  const [ canceler ] = useState(new AbortController());

  useEffect(() => {
    consumeStream(
      detApi.StreamingInternal.determinedTrialsSample(
        experiment.id,
        metric.name,
        metricTypeParamMap[metric.type],
        MAX_TRIALS,
        MAX_DATAPOINTS,
        undefined,
        undefined,
        undefined,
        { signal: canceler.signal },
      ),
      event => {
        console.log('event', event);
      },
    );

    return () => canceler.abort();
  }, [ canceler, experiment.id, metric ]);

  return (
    <Section title="Learning Curve">
      <div className={css.base}>
        <LearningCurveChart />
      </div>
    </Section>
  );
};

export default LearningCurve;
