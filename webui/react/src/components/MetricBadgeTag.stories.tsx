import React from 'react';

import { Metric, MetricType } from 'types';

import MetricBadgeTag from './MetricBadgeTag';

export default {
  component: MetricBadgeTag,
  title: 'Determined/Badges/MetricBadgeTag',
};

const trainingMetric: Metric = {
  name: 'training_accuracy',
  type: MetricType.Training,
};

const validationMetric: Metric = {
  name: 'validation_accuracy',
  type: MetricType.Validation,
};

export const Training = (): React.ReactNode => <MetricBadgeTag metric={trainingMetric} />;

export const Validation = (): React.ReactNode => <MetricBadgeTag metric={validationMetric} />;
