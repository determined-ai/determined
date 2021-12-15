import { MetricType, WorkloadGroup } from 'types';

import * as utils from './metric';

const workloads: WorkloadGroup[] = [
  {
    training: {
      metrics: { accuracy: 0.9, loss: 0.1 },
      totalBatches: 100,
    },
  },
  {
    training: {
      metrics: { accuracy: 0.91, loss: 0.09 },
      totalBatches: 200,
    },
    validation: {
      metrics: { accuracy: 0.81, loss: 0.19 },
      totalBatches: 200,
    },
  },
];

const metricNames = [
  {
    metric: { name: 'accuracy', type: MetricType.Training },
    str: '[T] accuracy',
    value: `${MetricType.Training}|accuracy`,
  },
  {
    metric: { name: 'loss', type: MetricType.Training },
    str: '[T] loss',
    value: `${MetricType.Training}|loss`,
  },
  {
    metric: { name: 'accuracy', type: MetricType.Validation },
    str: '[V] accuracy',
    value: `${MetricType.Validation}|accuracy`,
  },
  {
    metric: { name: 'loss', type: MetricType.Validation },
    str: '[V] loss',
    value: `${MetricType.Validation}|loss`,
  },
];

describe('Metric Utilities', () => {
  describe('extractMetricNames', () => {
    it('should extract metric names from workloads', () => {
      const result = [
        { name: 'accuracy', type: MetricType.Validation },
        { name: 'loss', type: MetricType.Validation },
        { name: 'accuracy', type: MetricType.Training },
        { name: 'loss', type: MetricType.Training },
      ];
      expect(utils.extractMetricNames(workloads)).toStrictEqual(result);
    });
  });

  describe('extractMetricValue', () => {
    const accuracyTraining = metricNames[0].metric;
    const lossValidation = metricNames[3].metric;

    it('should extract training metric', () => {
      expect(utils.extractMetricValue(workloads[0], accuracyTraining)).toBe(0.9);
    });

    it('should extract validation metric', () => {
      expect(utils.extractMetricValue(workloads[1], lossValidation)).toBe(0.19);
    });

    it('should handle non-existent metric extraction', () => {
      expect(utils.extractMetricValue(workloads[0], lossValidation)).toBeUndefined();
    });
  });

  describe('getMetricValue', () => {
    const workload = { metrics: { abc: 123 } };

    it('should return metric value when available', () => {
      expect(utils.getMetricValue(workload, 'abc')).toBe(123);
    });

    it('should return metric value when not available', () => {
      expect(utils.getMetricValue(workload, 'def')).toBeUndefined();
    });

    it('should return `undefined` if input is not a valid workload', () => {
      expect(utils.getMetricValue({})).toBeUndefined();
    });

    it('should return `undefined` if input metric name is invalid', () => {
      expect(utils.getMetricValue(workload)).toBeUndefined();
    });
  });

  describe('metricNameToStr', () => {
    it('should convert metric to string', () => {
      metricNames.forEach(metricName => {
        expect(utils.metricNameToStr(metricName.metric)).toBe(metricName.str);
      });
    });

    it('should truncate metric string to 30 characters', () => {
      const metricName = {
        name: 'very-very-very-very-very-very-long-metric-name',
        type: MetricType.Training,
      };
      expect(utils.metricNameToStr(metricName, 20)).toBe('[T] very-very-very-very-...');
    });
  });

  describe('metricNameToValue', () => {
    it('should convert metric to value', () => {
      metricNames.forEach(metricName => {
        expect(utils.metricNameToValue(metricName.metric)).toBe(metricName.value);
      });
    });
  });

  describe('valueToMetricName', () => {
    it('should convert value to metric name', () => {
      metricNames.forEach(metricName => {
        expect(utils.valueToMetricName(metricName.value)).toStrictEqual(metricName.metric);
      });
    });

    it('should handle invalid metric name value', () => {
      expect(utils.valueToMetricName('invalidMetricValue')).toBeUndefined();
      expect(utils.valueToMetricName('fauxMetricType|loss')).toBeUndefined();
    });
  });
});
