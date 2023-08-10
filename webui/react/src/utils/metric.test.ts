import { MetricType, WorkloadGroup } from 'types';

import * as utils from './metric';

const workloads: WorkloadGroup[] = [
  {
    metrics: {
      training: {
        metrics: { accuracy: 0.9, loss: 0.1 },
        totalBatches: 100,
      },
    },
  },
  {
    metrics: {
      training: {
        metrics: { accuracy: 0.91, loss: 0.09 },
        totalBatches: 200,
      },
      validation: {
        metrics: { accuracy: 0.81, loss: 0.19 },
        totalBatches: 200,
      },
    },
  },
];

const metrics = [
  {
    metric: { group: MetricType.Training, name: 'accuracy' },
    str: '[T] accuracy',
    value: '{"group":"training","name":"accuracy"}',
  },
  {
    metric: { group: MetricType.Training, name: 'loss' },
    str: '[T] loss',
    value: '{"group":"training","name":"loss"}',
  },
  {
    metric: { group: MetricType.Validation, name: 'accuracy' },
    str: '[V] accuracy',
    value: '{"group":"validation","name":"accuracy"}',
  },
  {
    metric: { group: MetricType.Validation, name: 'loss' },
    str: '[V] loss',
    value: '{"group":"validation","name":"loss"}',
  },
];

describe('Metric Utilities', () => {
  describe('extractMetrics', () => {
    it('should extract metric names from workloads', () => {
      const result = [
        { group: MetricType.Training, name: 'accuracy' },
        { group: MetricType.Training, name: 'loss' },
        { group: MetricType.Validation, name: 'accuracy' },
        { group: MetricType.Validation, name: 'loss' },
      ];
      expect(utils.extractMetrics(workloads)).toStrictEqual(result);
    });
  });

  describe('extractMetricValue', () => {
    const accuracyTraining = metrics[0].metric;
    const lossValidation = metrics[3].metric;

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

  describe('metricToStr', () => {
    it('should convert metric to string', () => {
      metrics.forEach((metric) => {
        expect(utils.metricToStr(metric.metric)).toBe(metric.str);
      });
    });

    it('should truncate metric string to 30 characters', () => {
      const metric = {
        group: MetricType.Training,
        name: 'very-very-very-very-very-very-long-metric-name',
      };
      expect(utils.metricToStr(metric, 20)).toBe('[T] very-very-very-very-...');
    });
  });

  describe('metricToKey', () => {
    it('should convert metric to value', () => {
      metrics.forEach((metric) => {
        expect(utils.metricToKey(metric.metric)).toBe(metric.value);
      });
    });
  });

  describe('metricKeyToMetric', () => {
    it('should convert value to metric name', () => {
      metrics.forEach((metric) => {
        expect(utils.metricKeyToMetric(metric.value)).toStrictEqual(metric.metric);
      });
    });

    it('should handle invalid metric name value', () => {
      expect(utils.metricKeyToMetric('invalidMetricValue')).toEqual({
        group: 'invalidMetricValue',
        name: 'invalidMetricValue',
      });
    });
  });
});
