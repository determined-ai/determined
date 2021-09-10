export enum MetricType {
  System = 'PROFILER_METRIC_TYPE_SYSTEM',
  Throughput = 'PROFILER_METRIC_TYPE_MISC',
  Timing = 'PROFILER_METRIC_TYPE_TIMING',
}

// {[metric_type]: {[name]: {[agent]: [gpu, ..], ..}, ..}, ..}
export type AvailableSeriesType = Record<string, Record<string, string[]>>;
export type AvailableSeries = Record<string, AvailableSeriesType>;

export type MetricsAggregateInterface = {
  // group information by { [time]: { [name]: value, ... }, ... }
  dataByTime: Record<number, Record<string, number>>,
  isEmpty: boolean,
  // set to false when the 1st event is received
  isLoading: boolean,
  // names to ease building the chart later
  names: string[],
};
