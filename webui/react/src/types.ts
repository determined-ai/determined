export interface User {
  id: number;
  isActive: boolean;
  isAdmin: boolean;
  username: string;
}

export interface Auth {
  isAuthenticated: boolean;
  user?: User;
}

export interface Credentials {
  password?: string;
  username: string;
}

export interface DeterminedInfo {
  clusterId: string;
  masterId: string;
  telemetry: {
    enabled: boolean;
    segmentKey?: string;
  };
  version: string;
}

export enum ResourceType {
  CPU = 'CPU',
  GPU = 'GPU'
}

export const resourceTypes: ResourceType[] = [
  ResourceType.CPU,
  ResourceType.GPU,
];

export enum ResourceState { // This is almost CommandState
  Free = 'FREE',
  Assigned = 'ASSIGNED',
  Pulling = 'PULLING',
  Starting = 'STARTING',
  Running = 'RUNNING',
  Terminating = 'TERMINATING',
  Terminated = 'TERMINATED',
}

export const resourceStates: ResourceState[] = [
  ResourceState.Free,
  ResourceState.Assigned,
  ResourceState.Pulling,
  ResourceState.Starting,
  ResourceState.Running,
  ResourceState.Terminating,
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

export interface Owner {
  id: number;
  username: string;
}

export enum CommandType {
  Command = 'COMMAND',
  Notebook = 'NOTEBOOK',
  Shell = 'SHELL',
  Tensorboard = 'TENSORBOARD',
}

export interface CommandMisc {
  experimentIds?: number[];
  trialIds?: number[];
}

export interface CommandConfig {
  description: string;
}

// The command type is shared between Commands, Notebooks, Tensorboards, and Shells.
export interface Command {
  kind: CommandType;
  config: CommandConfig; // We do not use this field in the WebUI.
  exitStatus?: string;
  id: string;
  misc?: CommandMisc;
  owner: Owner;
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
  Canceled = 'CANCELED',
  Completed = 'COMPLETED',
  Deleted = 'DELETED',
  Errored = 'ERROR',
  Paused = 'PAUSED',
  StoppingCanceled = 'STOPPING_CANCELED',
  StoppingCompleted = 'STOPPING_COMPLETED',
  StoppingError = 'STOPPING_ERROR',
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
}

export interface MetricNames {
  training: string[];
  validation: string[];
}

interface StepBase {
  id: number;
  trialId: number;
}

// Checkpoint sub step.
export interface Checkpoint extends StepBase, StartEndTimes {
  resources: Record<string, number>;
  state: CheckpointState;
  stepId: number;
  uuid? : string;
  validationMetric? : number;
}

// Validation sub step.
export interface Validation extends StepBase, StartEndTimes {
  state: RunState;
  stepId: number;
  metrics?: ValidationMetrics;
}

export interface Step extends StepBase, StartEndTimes {
  avgMetrics?: Record<string, number>;
  checkpoint?: Checkpoint;
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

interface TrialBase extends StartEndTimes {
  experimentId: number;
  id: number;
  state: RunState;
  seed: number;
  hparams: Record<string, string>;
}

export interface TrialItem extends TrialBase {
  bestAvailableCheckpoint?: Checkpoint;
  bestValidationMetric?: number;
  latestValidationMetrics?: ValidationMetrics;
  numBatches: number;
  numCompletedCheckpoints: number;
  numSteps: number;
  url: string;
}

export interface TrialDetails extends TrialBase {
  steps: Step[];
  warmStartCheckpointId?: number;
}

export interface Experiment {
  archived: boolean;
  config: ExperimentConfig;
  configRaw: Record<string, unknown>; // Readonly unparsed config object.
  endTime?: string;
  id: number;
  ownerId: number;
  progress?: number;
  startTime: string;
  state: RunState;
}

export interface ExperimentItem extends Experiment {
  name: string;
  url: string;
  username: string;
}

export interface ExperimentDetails extends Experiment {
  validationHistory: ValidationHistory[];
  trials: TrialItem[];
  username: string;
}

export interface Task {
  name: string;
  id: string;
  ownerId: number;
  url?: string;
  startTime: string;
}

export interface ExperimentTask extends Task {
  progress?: number;
  archived: boolean;
  state: RunState;
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
  limit: number;
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
