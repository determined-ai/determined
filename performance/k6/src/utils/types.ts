export interface Authorization {
  token: string;
}

export interface Experiment {
  id: string;
  metricName: string;
  metricType: string;
  batches: string;
  batchesMargin: string;
}

export interface Task {
  id: string;
}

export interface TestConfiguration {
  auth: Authorization;
  seededData: SeededData;
}

export interface TestGroup {
  name: string;
  group: () => void;
  enabled?: boolean;
}

interface Trial {
  id: string;
}

interface Model {
  name: string;
  versionNum: string;
}

export interface SeededData {
  model: Model;
  task: Task;
  trial: Trial;
  experiment: Experiment;
  workspace: Workspace;
  resourcePool: string;
}

interface Workspace {
  id: string;
  projectId: string;
}

interface Check {
  name: string;
  path: string;
  id: string;
  passes: number;
  fails: number;
}
export interface ResultGroup {
  name: string;
  path: string;
  id: string;
  group: ResultGroup[];
  checks: Check[];
}

export interface MetricResults {
  type: string;
  contains: string;
  values: Stats;
  thresholds: Thresholds;
}
interface Metric {
  [name: string]: MetricResults;
}

interface Stat {
  avg: number;
  min: number;
  med: number;
  max: number;
  "p(90)": number;
  "p(95)": number;
}
interface Stats {
  [name: string]: Stat;
}

interface ThresholdResults {
  ok: boolean;
}

interface Thresholds {
  [name: string]: ThresholdResults;
}

export interface Results {
  root_group: ResultGroup;
  metrics: Metric;
}
