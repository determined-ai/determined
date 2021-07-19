import {
  V1FittingPolicy, V1Pagination, V1ResourcePoolType, V1SchedulerType,
} from 'services/api-ts-sdk';

interface WithPagination {
  pagination: V1Pagination;
}

export type RecordKey = string | number | symbol;
export type UnknownRecord = Record<RecordKey, unknown>;
export type Primitive = boolean | number | string;
export type Point = { x: number; y: number };
export type Range<T = Primitive> = [ T, T ];

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type RawJson = Record<string, any>;

export type PropsWithClassName<T> = T & { className?: string };
export type PropsWithStoragePath<T> = T & { storagePath?: string };

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

export const deviceTypes = new Set([ ResourceType.CPU, ResourceType.GPU ]);

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

export interface NotebookConfig {
  name?: string;
  pool?: string;
  slots?: number;
  template?: string;
}

export enum CheckpointStorageType {
  AWS = 'aws',
  GCS = 'gcs',
  HDFS = 'hdfs',
  S3 = 's3',
  AZURE = 'azure',
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

export type HpImportance = Record<string, number>;
export type HpImportanceMetricMap = Record<string, HpImportance>;
export type HpImportanceMap = { [key in MetricType]: HpImportanceMetricMap };

export enum HyperparameterType {
  Categorical = 'categorical',
  Constant = 'const',
  Double = 'double',
  Int = 'int',
  Log = 'log',
}

export interface HyperparameterBase {
  base?: number;
  count?: number;
  maxval?: number;
  minval?: number;
  type?: HyperparameterType;
  val?: Primitive | Hyperparameters;
  vals?: Primitive[];
}

export interface Hyperparameter extends Omit<HyperparameterBase, 'type' | 'val'> {
  type: HyperparameterType;
  val?: Primitive;
}

export type Hyperparameters = {
  [keys: string]: Hyperparameters | HyperparameterBase;
};

/*
 * Flattened type for nested hyperparameters for easier WebUI usage and consumption.
 * The nested hyperparameters config currently come through as an implicit nested
 * dictionary and the way to distinguish a categorical type from a nested hp is the
 * detected of the property type. Where type is undefined for an implicit dictionary,
 * otherwise it is a terminal property where it has hp config info.
 */
export type HyperparametersFlattened = {
  [keys: string]: Hyperparameter;
};

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
  description?: string;
  hyperparameters: Hyperparameters;
  labels?: string[];
  name: string;
  profiling?: {
    enabled: boolean;
  };
  resources: {
    maxSlots?: number;
  };
  searcher: {
    max_trials?: number;
    metric: string;
    name: ExperimentSearcherName;
    smallerIsBetter: boolean;
  };
}

/* Experiment */

export enum ExperimentAction {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  CompareTrials = 'Compare Trials',
  ContinueTrial = 'Continue Trial',
  Delete = 'Delete',
  Fork = 'Fork',
  Kill = 'Kill',
  Pause = 'Pause',
  OpenTensorBoard = 'View in TensorBoard',
  Unarchive = 'Unarchive',
  ViewLogs = 'View Logs',
}

export interface ExperimentPagination extends WithPagination {
  experiments: ExperimentItem[];
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
  Deleting = 'DELETING',
  DeleteFailed = 'DELETE_FAILED',
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
  trialId: number;
  uuid? : string;
  validationMetric? : number;
}

export interface Workload extends StartEndTimes {
  totalBatches: number;
}

export interface CheckpointWorkload extends Workload {
  resources?: Record<string, number>;
  uuid? : string;
}

export interface CheckpointWorkloadExtended extends CheckpointWorkload {
  experimentId: number;
  trialId: number;
}

export interface MetricsWorkload extends Workload {
  metrics?: Record<string, number>;
  numInputs?: number;
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

type HpValue = Primitive | RawJson
export type TrialHyperparameters = Record<string, HpValue>

export interface TrialItem extends StartEndTimes {
  bestAvailableCheckpoint?: CheckpointWorkload;
  bestValidationMetric?: MetricsWorkload;
  experimentId: number;
  hyperparameters: TrialHyperparameters;
  id: number;
  latestValidationMetric?: MetricsWorkload;
  state: RunState;
  totalBatchesProcessed: number;
}

export interface TrialDetails extends TrialItem {
  runnerState?: string;
  workloads: WorkloadWrapper[];
}

export interface ExperimentItem {
  archived: boolean;
  description?: string;
  endTime?: string;
  id: number;
  labels: string[];
  name: string;
  numTrials: number;
  progress?: number;
  resourcePool: string
  startTime: string;
  state: RunState;
  username: string;
}

export interface ExperimentBase {
  archived: boolean;
  config: ExperimentConfig;
  configRaw: RawJson;                                 // Readonly unparsed config object.
  description?: string;
  endTime?: string;
  hyperparameters: HyperparametersFlattened;    // nested hp keys are flattened, eg) foo.bar
  id: number;
  name: string;
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

export enum ExperimentVisualizationType {
  HpParallelCoordinates = 'hp-parallel-coordinates',
  HpHeatMap = 'hp-heat-map',
  HpScatterPlots = 'hp-scatter-plots',
  LearningCurve = 'learning-curve',
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

export type TaskType = CommandType | 'Experiment';

export enum ArchiveFilter {
  Archived = 'archived',
  Unarchived = 'unarchived',
}

export interface ExperimentFilters {
  archived?: ArchiveFilter;
  labels?: string[];
  states?: string[];
  users?: string[];
}

export interface TrialFilters {
  states?: string[];
}

export interface TaskFilters<T extends TaskType = TaskType> {
  limit: number;
  states?: string[];
  types?: T[];
  users?: string[];
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

export interface Template {
  config?: RawJson;
  name: string;
}

export interface ResourcePool {
  auxContainerCapacity: number;
  auxContainerCapacityPerAgent: number;
  auxContainersRunning: number;
  defaultAuxPool: boolean;
  defaultComputePool?: boolean;
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
  slotType: ResourceType;
  slotsAvailable: number;
  slotsPerAgent?: number;
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
