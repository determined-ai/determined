import { MetricName, MetricType, Step } from 'types';
import { isNumber, metricNameSorter } from 'utils/data';

import handleError, { DaError, ErrorLevel, ErrorType } from '../ErrorHandler';

export const extractMetricValue = (step: Step, metricName: MetricName): number | undefined => {
  if (metricName.type === MetricType.Training) {
    const source = step.avgMetrics || {};
    if (isNumber(source[metricName.name])) return source[metricName.name];
  } else if (metricName.type === MetricType.Validation) {
    const source = step.validation?.metrics?.validationMetrics || {};
    if (isNumber(source[metricName.name])) return source[metricName.name];
  }
  return undefined;
};

export const extractMetricNames = (steps: Step[] = []): MetricName[] => {
  const map: Record<string, MetricName> = {};

  steps.forEach(step => {
    const trainingSource = step.avgMetrics || {};
    const validationSource = step.validation?.metrics?.validationMetrics || {};

    // Extract training metric names
    Object.keys(trainingSource).forEach(key => {
      if (!isNumber(trainingSource[key])) return;
      const metricName = { name: key, type: MetricType.Training };
      const value = metricNameToValue(metricName);
      if (!map[value]) map[value] = metricName;
    });

    // Extract validation metric names
    Object.keys(validationSource).forEach(key => {
      if (!isNumber(validationSource[key])) return;
      const metricName = { name: key, type: MetricType.Validation };
      const value = metricNameToValue(metricName);
      if (!map[value]) map[value] = metricName;
    });
  });

  return Object.values(map).sort(metricNameSorter);
};

export const metricNameToValue = (metricName: MetricName): string => {
  return `${metricName.type}|${metricName.name}`;
};

export const metricNameFromValue = (metricValue: string): MetricName | undefined => {
  const trainingPrefix = `${MetricType.Training}|`;
  const validationPrefix = `${MetricType.Validation}|`;
  if (metricValue.startsWith(trainingPrefix)) {
    return {
      name: metricValue.slice(trainingPrefix.length),
      type: MetricType.Training,
    };
  } else if (metricValue.startsWith(validationPrefix)) {
    return {
      name: metricValue.slice(validationPrefix.length),
      type: MetricType.Validation,
    };
  } else {
    const errName = 'metricNameFromValueUnrecognizedMetricType';
    const errSlug = 'metricnamefromvalue-unrecognized-metric-type';
    const errMessage = `
      metricNameFromValue was called, but the metricName doesn't appear to 
      be a training metric or a validation metric (${metricValue})
    `;

    const daErr: DaError = {
      error: {
        message: errMessage,
        name: errName,
      },
      id: errSlug,
      isUserTriggered: false,
      level: ErrorLevel.Error,
      message: errMessage,
      silent: !process.env.IS_DEV,
      type: ErrorType.Ui,
    };
    handleError(daErr);
    return undefined;
  }
};

export const valueToMetricName = (value: string): MetricName | undefined => {
  const parts = value.split('|');
  if (parts.length === 2) return { name: parts[1], type: parts[0] as MetricType };
  return undefined;
};
