import { MetricName, MetricType, Step } from 'types';
import { isNumber, metricNameSorter } from 'utils/data';

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

export const valueToMetricName = (value: string): MetricName | undefined => {
  const parts = value.split('|');
  if (parts.length === 2) return { name: parts[1], type: parts[0] as MetricType };
  return undefined;
};
