import { Serie } from 'components/kit/LineChart';
import {
  V1GetTrialProfilerMetricsResponse,
  V1TrialProfilerMetricsBatch,
} from 'services/api-ts-sdk';
import { TrialDetails, ValueOf } from 'types';

export const MetricType = {
  System: 'PROFILER_METRIC_TYPE_SYSTEM',
  Throughput: 'PROFILER_METRIC_TYPE_MISC',
  Timing: 'PROFILER_METRIC_TYPE_TIMING',
} as const;

export type MetricType = ValueOf<typeof MetricType>;

// {[metric_type]: {[name]: {[agent]: [gpu, ..], ..}, ..}, ..}
export type AvailableSeriesType = Record<string, Record<string, string[]>>;
export type AvailableSeries = Record<string, AvailableSeriesType>;

export type MetricsAggregateInterface = {
  // group information by { [time]: { [name]: value, ... }, ... }
  data: Serie[];
  initialTimestamp?: number;
  isEmpty: boolean;
  // set to false when the 1st event is received
  isLoading: boolean;
  names: string[];
};

export type OldMetricsAggregateInterface = {
  // group information by { [time]: { [name]: value, ... }, ... }
  data?: uPlot.AlignedData;
  initialTimestamp?: number;
  isEmpty: boolean;
  // set to false when the 1st event is received
  isLoading: boolean;
  names: string[];
};

export interface ChartProps {
  trial: TrialDetails;
}

export interface ProfilerMetricsBatch extends Omit<V1TrialProfilerMetricsBatch, 'timestamps'> {
  timestamps: string[];
}

export interface ProfilerMetricsResponse extends Omit<V1GetTrialProfilerMetricsResponse, 'batch'> {
  batch: ProfilerMetricsBatch;
}
