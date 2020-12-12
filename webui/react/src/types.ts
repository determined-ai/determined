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
  token?: string;
  isAuthenticated: boolean;
  user?: User;
}

export interface Credentials {
  password?: string;
  username: string;
}

export interface DeterminedInfo {
  clusterId: string;
  clusterName: string;
  masterId: string;
  telemetry: {
    enabled: boolean;
    segmentKey?: string;
  };
  version: string;
}

export enum ResourceType {
  CPU = 'CPU',
  GPU = 'GPU',
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
  id: string;
  name: string;
  uuid?: string;
  type: ResourceType;
  enabled: boolean;
  container?: ResourceContainer;
}

export interface Agent {
  id: string;
  registeredTime: number;
  resources: Resource[];
}

export interface ClusterOverviewResource {
  available: number;
  total: number;
}

export interface ClusterOverview {
  [ResourceType.CPU]: ClusterOverviewResource;
  [ResourceType.GPU]: ClusterOverviewResource;
  allocation: number;
  totalResources: ClusterOverviewResource;
}

export interface StartEndTimes {
  endTime?: string;
  startTime: string;
}

export interface Pagination {
  offset: number;
  limit: number;
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
  kind: CommandType; // TODO rename to type
  config: CommandConfig; // We do not use this field in the WebUI.
  exitStatus?: string;
  id: string;
  misc?: CommandMisc;
  user: User;
  registeredTime: string;
  serviceAddress?: string;
  state: CommandState;
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
  type: string;
  val: unknown;
}

export type ExperimentHyperParams = Record<string, ExperimentHyperParam>;

export interface ExperimentConfig {
  checkpointPolicy: string;
  checkpointStorage?: CheckpointStorage;
  dataLayer?: DataLayer;
  description: string;
  searcher: {
    smallerIsBetter: boolean;
    metric: string;
  };
  resources: {
    maxSlots?: number;
  };
  labels?: string[];
}

/* Experiment */
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

export interface MetricName {
  name: string;
  type: MetricType;
}

// Checkpoint sub step.
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

export interface MetricsWorkload extends Workload {
  state: RunState;
  metrics?: Record<string, number>;
  numInputs?: number;
}
export interface WorkloadWrapper {
  training?: MetricsWorkload;
  validation?: MetricsWorkload;
  checkpoint?: CheckpointWorkload;
}

// Validation sub step.
export interface Validation extends StartEndTimes {
  id: number;
  state: RunState;
  metrics?: ValidationMetrics;
}

export interface Step2 extends WorkloadWrapper, StartEndTimes {
  training: MetricsWorkload;
  batchNum: number;
}

export interface Step extends StartEndTimes {
  avgMetrics?: Record<string, number>;
  checkpoint?: Checkpoint;
  id: number;
  numBatches: number;
  priorBatchesProcessed: number;
  state: RunState;
  validation?: Validation;
}

export interface CheckpointDetail extends Checkpoint {
  batch: number;
  experimentId?: number;
}

export interface ValidationMetrics {
  numInputs: number;
  validationMetrics: Record<string, number>;
}

type HyperparameterValue = number | string | boolean | RawJson
export type TrialHyperParameters = Record<string, HyperparameterValue>

interface TrialBase extends StartEndTimes {
  experimentId: number;
  id: number;
  state: RunState;
  hparams: TrialHyperParameters;
}

// To replace TrialItem once experiment endpoint is migrated.
export interface TrialItem2 extends TrialBase {
  bestAvailableCheckpoint?: CheckpointWorkload;
  bestValidationMetric?: MetricsWorkload;
  latestValidationMetric?: MetricsWorkload;
  totalBatchesProcessed: number;
}

export interface TrialDetails2 extends TrialItem2 {
  workloads: WorkloadWrapper[];
}

export interface TrialItem extends TrialBase {
  bestAvailableCheckpoint?: Checkpoint;
  bestValidationMetric?: number;
  latestValidationMetrics?: ValidationMetrics;
  numCompletedCheckpoints: number;
  numSteps: number;
  totalBatchesProcessed: number;
  url: string;
  seed: number;
}

export interface ExperimentItem {
  id: number;
  name: string;
  labels: string[];
  startTime: string;
  endTime?: string;
  state: RunState;
  archived: boolean;
  numTrials: number;
  progress?: number;
  username: string;
  url: string;
}

export interface ExperimentBase {
  archived: boolean;
  config: ExperimentConfig;
  configRaw: RawJson; // Readonly unparsed config object.
  endTime?: string;
  id: number;
  username: string;
  progress?: number;
  startTime: string;
  state: RunState;
}

export interface ExperimentOld extends ExperimentBase {
  name: string;
  url: string;
  username: string;
}

export interface ExperimentDetails extends ExperimentBase {
  validationHistory: ValidationHistory[];
  trials: TrialItem[];
  username: string;
}

export interface Task {
  name: string;
  id: string;
  url?: string;
  serviceAddress?: string;
  startTime: string;
}

export interface ExperimentTask extends Task {
  progress?: number;
  archived: boolean;
  state: RunState;
  username: string;
}

export interface CommandTask extends Task {
  misc?: CommandMisc;
  type: CommandType;
  state: CommandState;
  username: string;
}

export type RecentEvent = {
  lastEvent: {
    name: string;
    date: string;
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
  showArchived: boolean;
  labels?: string[];
  states: string[];
  username?: string;
}

export interface TaskFilters<T extends TaskType = TaskType> {
  limit: number;
  states: string[];
  username?: string;
  types: Record<T, boolean>;
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
  className?: string;
  children?: React.ReactNode;
  title?: string;
};

export enum LogLevel {
  Debug = 'debug',
  Error = 'error',
  Info = 'info',
  Warning = 'warning',
}

export interface Log {
  id: number;
  level?: LogLevel;
  message: string;
  meta?: string;
  time?: string;
}
