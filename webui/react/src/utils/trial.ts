import { MetricName, MetricType, WorkloadWrapper } from 'types';
import { isNumber, metricNameSorter } from 'utils/data';

import handleError, { DaError, ErrorLevel, ErrorType } from '../ErrorHandler';

import { getDuration } from './time';

export const extractMetricNames = (workloads: WorkloadWrapper[]): MetricName[] => {
  const trainingNames: Set<string> = workloads
    .filter(wl => wl.training?.metrics)
    .reduce((acc, cur) => {
      Object.keys(cur.training?.metrics as Record<string, number>).forEach(name => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>()) as Set<string>; // this "as" shouldn't be needed.

  const trainingMetrics: MetricName[]= Array.from(trainingNames).map(name => ({
    name,
    type: MetricType.Training,
  }));

  const validationNames: Set<string> = workloads
    .filter(wl => wl.validation?.metrics)
    .reduce((acc, cur) => {
      Object.keys(cur.validation?.metrics as Record<string, number>).forEach(name => {
        acc.add(name);
      });
      return acc;
    }, new Set<string>()) as Set<string>; // this "as" shouldn't be needed.

  const validationMetrics: MetricName[]= Array.from(validationNames).map(name => ({
    name,
    type: MetricType.Validation,
  }));

  return [ ...validationMetrics, ...trainingMetrics ].sort(metricNameSorter);
};
export const metricNameToValue = (metricName: MetricName): string => {
  return `${metricName.type}|${metricName.name}`;
};

export const extractMetricValue = (
  wl: WorkloadWrapper,
  metricName: MetricName,
): number | undefined => {
  const source = (metricName.type === MetricType.Training
    ? wl.training?.metrics : wl.validation?.metrics) || {};
  if (isNumber(source[metricName.name])) return source[metricName.name];
  return undefined;
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

interface TrialDurations {
  checkpoint: number;
  train: number;
  validation: number;
}

export const trialDurations = (wlWrappers: WorkloadWrapper[]): TrialDurations => {
  const initialDurations: TrialDurations = {
    checkpoint: 0,
    train: 0,
    validation: 0,
  };

  return wlWrappers.reduce((acc: TrialDurations, cur: WorkloadWrapper) => {
    if (cur.training) acc.train += getDuration(cur.training);
    if (cur.checkpoint) acc.checkpoint += getDuration(cur.checkpoint);
    if (cur.validation) acc.validation += getDuration(cur.validation);
    return acc;
  }, initialDurations);
};
