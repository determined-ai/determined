import { MetricNames, Step } from 'types';
import { isNumber } from 'utils/data';

export const extractMetricNames = (steps: Step[] = []): MetricNames => {
  const trainingMap: Record<string, boolean> = {};
  const validationMap: Record<string, boolean> = {};

  steps.forEach(step => {
    const trainingSource = step.avgMetrics || {};
    const validationSource = step.validation?.metrics?.validationMetrics || {};

    // Extract training metric names
    Object.keys(trainingSource).forEach(key => {
      if (!isNumber(trainingSource[key])) return;
      trainingMap[key] = true;
    });

    // Extract validation metric names
    Object.keys(validationSource).forEach(key => {
      if (!isNumber(validationSource[key])) return;
      validationMap[key] = true;
    });
  });

  return {
    training: Object.keys(trainingMap).sort(),
    validation: Object.keys(validationMap).sort(),
  };
};
