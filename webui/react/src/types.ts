import {
  V1FittingPolicy, V1Pagination, V1ResourcePoolType, V1SchedulerType,
} from 'services/api-ts-sdk';

interface WithPagination {
  pagination: V1Pagination;
}

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type RawJson = Record<string, any>;

export interface User {
  username: string;
}

export interface DetailedUser extends User {
  isActive: boolean;
  isAdmin: boolean;
}

export interface Auth {
  isAuthenticated: boolean;
  token?: string;
  user?: DetailedUser;
}

export interface Credentials {
  password?: string;
  username: string;
}

export interface DeterminedInfo {
  clusterId: string;
  clusterName: string;
  isTelemetryEnabled: boolean;
  masterId: string;
  version: string;
}

export interface Telemetry {
  enabled: boolean;
  segmentKey?: string;
}

export enum ResourceType {
  CPU = 'CPU',
  GPU = 'GPU',
  ALL = 'ALL',
  UNSPECIFIED = 'UNSPECIFIED',
}

export const resourceTypes: ResourceType[] = [
  ResourceType.CPU,
  ResourceType.GPU,
];

export enum ResourceState { // This is almost CommandState
  Unspecified = 'UNSPECIFIED',
  Assigned = 'ASSIGNED',
  Pulling = 'PULLING',
  Starting = 'STARTING',
  Running = 'RUNNING',
  Terminated = 'TERMINATED',
}

// High level Slot state
export enum SlotState {
  Running = 'RUNNING',
  Free = 'FREE',
  Pending = 'PENDING',
}

export const resourceStates: ResourceState[] = [
  ResourceState.Unspecified,
  ResourceState.Assigned,
  ResourceState.Pulling,
  ResourceState.Starting,
  ResourceState.Running,
  ResourceState.Terminated,
];

export interface ResourceContainer {
  id: string;
  state: ResourceState;
}

export interface Resource {
  container?: ResourceContainer;
  enabled: boolean;
  id: string;
  name: string;
  type: ResourceType;
  uuid?: string;
}

export interface Agent {
  id: string;
  registeredTime: number;
  resourcePool: string;
  resources: Resource[];
}

export interface ClusterOverviewResource {
  allocation: number;
  available: number;
  total: number;
}

export type ClusterOverview = Record<ResourceType, ClusterOverviewResource>;

export interface StartEndTimes {
  endTime?: string;
  startTime: string;
}

export interface Pagination {
  limit: number;
  offset: number;
}

/* Command */
export enum CommandState {
  Pending = 'PENDING',
  Assigned = 'ASSIGNED',
  Pulling = 'PULLING',
  Starting = 'STARTING',
  Running = 'RUNNING',
  Terminating = 'TERMINATING',
  Terminated = 'TERMINATED',
}

export type State = CommandState | RunState;

export interface CommandAddress {
  containerIp: string;
  containerPort: number;
  hostIp: string;
  hostPort: number;
  protocol?: string;
}

export enum CommandType {
  Command = 'COMMAND',
  Notebook = 'NOTEBOOK',
  Shell = 'SHELL',
  Tensorboard = 'TENSORBOARD',
}

export interface CommandMisc {
  experimentIds: number[];
  trialIds: number[];
}

export interface CommandConfig {
  description: string;
}

// The command type is shared between Commands, Notebooks, Tensorboards, and Shells.
export interface Command {
  config: CommandConfig; // We do not use this field in the WebUI.
  exitStatus?: string;
  id: string;
  kind: CommandType; // TODO rename to type
  misc?: CommandMisc;
  registeredTime: string;
  resourcePool: string;
  serviceAddress?: string;
  state: CommandState;
  user: User;
}

export enum CheckpointStorageType {
  AWS = 'aws',
  GCS = 'gcs',
  HDFS = 'hdfs',
  S3 = 's3',
  SharedFS = 'shared_fs',
}

interface CheckpointStorage {
  bucket?: string;
  hostPath?: string;
  saveExperimentBest: number;
  saveTrialBest: number;
  saveTrialLatest: number;
  storagePath?: string;
  type?: CheckpointStorageType;
}

interface DataLayer {
  containerStoragePath?: string;
  type: string;
}

interface ExperimentHyperParam {
  base?: number;
  count?: number;
  maxval?: number;
  minval?: number;
  type: string;
  val?: unknown;
}

export type ExperimentHyperParams = Record<string, ExperimentHyperParam>;

export enum ExperimentSearcherName {
  AdaptiveAdvanced = 'adaptive',
  AdaptiveAsha = 'adaptive_asha',
  AdaptiveSimple = 'adaptive_simple',
  Grid = 'grid',
  Pbt = 'pbt',
  Random = 'random',
  Single = 'single',
}

export interface ExperimentConfig {
  checkpointPolicy: string;
  checkpointStorage?: CheckpointStorage;
  dataLayer?: DataLayer;
  description: string;
  hyperparameters: ExperimentHyperParams;
  labels?: string[];
  resources: {
    maxSlots?: number;
  };
  searcher: {
    metric: string;
    name: ExperimentSearcherName;
    smallerIsBetter: boolean;
  };
}

/* Experiment */

export interface ExperimentPagination extends WithPagination {
  experiments: ExperimentBase[];
}

export enum RunState {
  Active = 'ACTIVE',
  Paused = 'PAUSED',
  StoppingCanceled = 'STOPPING_CANCELED',
  Canceled = 'CANCELED',
  StoppingCompleted = 'STOPPING_COMPLETED',
  Completed = 'COMPLETED',
  StoppingError = 'STOPPING_ERROR',
  Errored = 'ERROR',
  Deleted = 'DELETED',
  Unspecified = 'UNSPECIFIED',
}

export interface ValidationHistory {
  endTime: string;
  trialId: number;
  validationError?: number;
}

export enum CheckpointState {
  Active = 'ACTIVE',
  Completed = 'COMPLETED',
  Error = 'ERROR',
  Deleted = 'DELETED',
  Unspecified = 'UNSPECIFIED',
}

export enum MetricType {
  Training = 'training',
  Validation = 'validation',
}

export type MetricTypeParam =
  'METRIC_TYPE_UNSPECIFIED' | 'METRIC_TYPE_TRAINING' | 'METRIC_TYPE_VALIDATION';

export const metricTypeParamMap: Record<string, MetricTypeParam> = {
  [MetricType.Training]: 'METRIC_TYPE_TRAINING',
  [MetricType.Validation]: 'METRIC_TYPE_VALIDATION',
};

export interface MetricName {
  name: string;
  type: MetricType;
}

export interface Checkpoint extends StartEndTimes {
  resources?: Record<string, number>;
  state: CheckpointState;
  trialId: number;
  uuid? : string;
  validationMetric? : number;
}

export interface Workload extends StartEndTimes {
  numBatches: number;
  priorBatchesProcessed: number;
}

export interface CheckpointWorkload extends Workload {
  resources?: Record<string, number>;
  state: CheckpointState;
  uuid? : string;
}

export interface CheckpointWorkloadExtended extends CheckpointWorkload {
  experimentId: number;
  trialId: number;
}

export interface MetricsWorkload extends Workload {
  metrics?: Record<string, number>;
  numInputs?: number;
  state: RunState;
}
export interface WorkloadWrapper {
  checkpoint?: CheckpointWorkload;
  training?: MetricsWorkload;
  validation?: MetricsWorkload;
}

// This is to support the steps table in trial details and shouldn't be used
// elsewhere so we can remove it with a redesign.
export interface Step extends WorkloadWrapper, StartEndTimes {
  batchNum: number;
  training: MetricsWorkload;
}

export interface CheckpointDetail extends Checkpoint {
  batch: number;
  experimentId?: number;
}

export interface ValidationMetrics {
  numInputs: number;
  validationMetrics: Record<string, number>;
}

export interface TrialPagination extends WithPagination {
  trials: TrialDetails[];
}

type HyperparameterValue = number | string | boolean | RawJson
export type TrialHyperParameters = Record<string, HyperparameterValue>

export interface TrialItem extends StartEndTimes {
  bestAvailableCheckpoint?: CheckpointWorkload;
  bestValidationMetric?: MetricsWorkload;
  experimentId: number;
  hparams: TrialHyperParameters;
  id: number;
  latestValidationMetric?: MetricsWorkload;
  state: RunState;
  totalBatchesProcessed: number;
}

export interface TrialDetails extends TrialItem {
  workloads: WorkloadWrapper[];
}

export interface ExperimentItem {
  archived: boolean;
  endTime?: string;
  id: number;
  labels: string[];
  name: string;
  numTrials: number;
  progress?: number;
  resourcePool: string
  startTime: string;
  state: RunState;
  url: string;
  username: string;
}

export interface ExperimentBase {
  archived: boolean;
  config: ExperimentConfig;
  configRaw: RawJson; // Readonly unparsed config object.
  endTime?: string;
  id: number;
  progress?: number;
  resourcePool: string;
  startTime: string;
  state: RunState;
  username: string;
}

export interface ExperimentOld extends ExperimentBase {
  name: string;
  url: string;
  username: string;
}

export interface Task {
  id: string;
  name: string;
  resourcePool: string;
  serviceAddress?: string;
  startTime: string;
  url?: string;
}

export interface ExperimentTask extends Task {
  archived: boolean;
  progress?: number;
  resourcePool: string;
  state: RunState;
  username: string;
}

export interface CommandTask extends Task {
  misc?: CommandMisc;
  resourcePool: string;
  state: CommandState;
  type: CommandType;
  username: string;
}

export type RecentEvent = {
  lastEvent: {
    date: string;
    name: string;
  };
};

export const ALL_VALUE = 'all';

export type AnyTask = CommandTask | ExperimentTask;
export type RecentTask = AnyTask & RecentEvent;
export type RecentCommandTask = CommandTask & RecentEvent;
export type RecentExperimentTask = ExperimentTask & RecentEvent;

export type PropsWithClassName<T> = T & {className?: string};

export type TaskType = CommandType | 'Experiment';

export interface ExperimentFilters {
  labels?: string[];
  showArchived: boolean;
  states: string[];
  username?: string;
}

export interface TaskFilters<T extends TaskType = TaskType> {
  limit: number;
  states: string[];
  types: Record<T, boolean>;
  username?: string;
}

export enum TBSourceType {
  Trial,
  Experiment
}

export interface TBSource {
  ids: number[];
  type: TBSourceType;
}

export type CommonProps = {
  children?: React.ReactNode;
  className?: string;
  title?: string;
};

export enum LogLevel {
  Critical = 'critical',
  Debug = 'debug',
  Error = 'error',
  Info = 'info',
  Trace = 'trace',
  Warning = 'warning',
}

export interface Log {
  id: number;
  level?: LogLevel;
  message: string;
  meta?: string;
  time?: string;
}

export interface TrialLog {
  id: string;
  level?: LogLevel;
  message: string;
  time: string;
}

export interface ResourcePool {
  cpuContainerCapacity: number;
  cpuContainerCapacityPerAgent: number;
  cpuContainersRunning: number;
  defaultCpuPool: boolean;
  defaultGpuPool?: boolean;
  description: string;
  details: RPDetails;
  imageId: string;
  instanceType: string;
  location: string;
  maxAgents: number;
  minAgents: number;
  name: string;
  numAgents: number;
  preemptible: boolean;
  schedulerFittingPolicy: V1FittingPolicy;
  schedulerType: V1SchedulerType;
  slotsAvailable: number;
  slotsUsed: number;
  type: V1ResourcePoolType;
}

export interface RPDetails {
  aws?: Partial<Aws>;
  gcp?: Partial<Gcp>;
  priorityScheduler?: PriorityScheduler;
}

export interface Aws {
  customTags?: CustomTag[];
  iamInstanceProfileArn: string;
  imageId: string;
  instanceName: string;
  instanceType: string;
  logGroup: string;
  logStream: string;
  publicIp: boolean;
  region: string;
  rootVolumeSize: number;
  securityGroupId: string;
  spotEnabled: boolean;
  spotMaxPrice: string;
  sshKeyName: string;
  subnetId: string;
  tagKey: string;
  tagValue: string;
}

interface CustomTag {
  key: string;
  value: string;
}

export interface Gcp {
  bootDiskSize: number;
  bootDiskSourceImage: string;
  externalIp: boolean;
  gpuNum: number;
  gpuType: string;
  labelKey: string;
  labelValue: string;
  machineType: string;
  namePrefix: string;
  network: string;
  networkTags: string[];
  operationTimeoutPeriod: number;
  preemptible: boolean;
  project: string;
  serviceAccountEmail: string;
  serviceAccountScopes: string[];
  subnetwork: string;
  zone: string;
}

export interface PriorityScheduler {
  defaultPriority: number;
  preemption: boolean;
}
