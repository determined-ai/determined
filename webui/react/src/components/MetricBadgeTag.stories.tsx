import React from 'react';

import { MetricName, MetricType } from 'types';

import MetricBadgeTag from './MetricBadgeTag';

export default {
  component: MetricBadgeTag,
  title: 'BadgeTag',
};

const trainingMetric: MetricName = {
  name: 'training_accuracy',
  type: MetricType.Training,
};

const validationMetric: MetricName = {
  name: 'validation_accuracy',
  type: MetricType.Validation,
};

export const Training = (): React.ReactNode => <MetricBadgeTag metric={trainingMetric} />;

export const Validation = (): React.ReactNode => <MetricBadgeTag metric={validationMetric} />;
