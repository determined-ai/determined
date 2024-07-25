import * as t from 'io-ts';
import { RouteProps } from 'react-router-dom';

import { DateString } from 'ioTypes';
import * as Api from 'services/api-ts-sdk';
import { V1AgentUserGroup, V1Group, V1LaunchWarning, V1Slot, V1Trigger } from 'services/api-ts-sdk';
import { valueof, ValueOf } from 'utils/valueof';

export type { ValueOf } from 'utils/valueof';
export const Primitive = t.union([t.boolean, t.number, t.string]);
export type Primitive = t.TypeOf<typeof Primitive>;
export type RecordKey = string | number | symbol;
export type UnknownRecord = Record<RecordKey, unknown>;
export type NullOrUndefined<T = undefined> = T | null | undefined;
export type Point = { x: number; y: number };
export type Range<T = Primitive> = [T, T];
export type Eventually<T> = T | Promise<T>;
type Without<T, U> = { [P in Exclude<keyof T, keyof U>]?: never };
// XOR is taken from: https://stackoverflow.com/a/53229857
export type XOR<T, U> = T | U extends object ? (Without<T, U> & U) | (Without<U, T> & T) : T | U;

// DEPRECATED
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type RawJson = Record<string, any>;

// these codecs have types defined because recursive types like these can't be inferred
export type JsonArray = Json[];
export const JsonArray: t.RecursiveType<t.Type<JsonArray>> = t.recursion('JsonArray', () =>
  t.array(Json),
);

export type JsonObject = {
  [key in string]: Json;
};
export const JsonObject: t.RecursiveType<t.Type<JsonObject>> = t.recursion('JsonObject', () =>
  t.record(t.string, Json),
);

export type Json = string | number | boolean | null | JsonArray | JsonObject;
export const Json = t.recursion<Json>('Json', () =>
  t.union([t.string, t.number, t.boolean, t.null, JsonArray, JsonObject]),
);

export interface Pagination {
  limit: number;
  offset: number;
  total?: number;
}

export interface FetchOptions {
  signal?: AbortSignal;
}

interface ApiBase {
  name: string;
  stubbedResponse?: unknown;
  unAuthenticated?: boolean;
  // middlewares?: Middleware[]; // success/failure middlewares
}

export type RecordUnknown = Record<RecordKey, unknown>;

// Designed for use with Swagger generated api bindings.
export interface DetApi<Input, DetOutput, Output> extends ApiBase {
  postProcess: (response: DetOutput) => Output;
  request: (params: Input, options?: FetchOptions) => Promise<DetOutput>;
  stubbedResponse?: DetOutput;
}

/**
 * @description helper to organize storing api response data.
 */
export interface ApiState<T> {
  data?: T;
  /**
   * error, if any, with the last state update.
   * this should be cleared on the next successful update.
   */
  error?: Error;
  /**
   * indicates whether the state has been fetched at least once or not.
   * should always be initialized to false.
   */
  hasBeenInitialized?: boolean;
  /** is the state being updated? */
  isLoading?: boolean;
}

export interface SingleEntityParams {
  id: number;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {};

/**
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export type RouteConfig = {
  icon?: string;
  id: string;
  needAuth?: boolean;
  path: string;
  popout?: boolean;
  redirect?: string;
  suffixIcon?: string;
  title?: string;
} & RouteProps;

export interface ClassNameProp {
  /** classname to be applied to the base element */
  className?: string;
}
export interface CommonProps extends ClassNameProp {
  children?: React.ReactNode;
  title?: string;
}

export interface SemanticVersion {
  major: number;
  minor: number;
  patch: number;
}

interface WithPagination {
  pagination: Api.V1Pagination; // probably should use this or Pagination
}

export type PropsWithStoragePath<T> = T & { storagePath?: string };
export const User = t.intersection([
  t.partial({
    displayName: t.string,
    lastAuthAt: t.number,
    modifiedAt: t.number,
  }),
  t.type({
    id: t.number,
    username: t.string,
  }),
]);
export type User = t.TypeOf<typeof User>;

const AgentUserGroup: t.Type<V1AgentUserGroup> = t.partial({
  agentGid: t.number,
  agentGroup: t.string,
  agentUid: t.number,
  agentUser: t.string,
});

export const DetailedUser = t.intersection([
  User,
  t.partial({
    agentUserGroup: AgentUserGroup,
    isPasswordWeak: t.boolean,
    remote: t.boolean,
  }),
  t.type({
    isActive: t.boolean,
    isAdmin: t.boolean,
  }),
]);
export type DetailedUser = t.TypeOf<typeof DetailedUser>;

export interface DetailedUserList extends WithPagination {
  users: DetailedUser[];
}

export interface Auth {
  isAuthenticated: boolean;
  token?: string;
}

// ResourceType key and value must be the same
export const ResourceType = {
  ALL: 'ALL',
  CPU: 'CPU',
  CUDA: 'CUDA',
  ROCM: 'ROCM',
  UNSPECIFIED: 'UNSPECIFIED',
} as const;

export type ResourceType = ValueOf<typeof ResourceType>;

export const isResourceType = (val: string): val is ResourceType => {
  return val in ResourceType;
};

export const isDeviceType = (type: ResourceType): boolean => {
  return ResourceType.CPU === type || ResourceType.CUDA === type || ResourceType.ROCM === type;
};

export const ResourceState = {
  // This is almost CommandState
  Assigned: 'ASSIGNED',
  Potential: 'POTENTIAL',
  Pulling: 'PULLING',
  Running: 'RUNNING',
  Starting: 'STARTING',
  Terminated: 'TERMINATED',
  Unspecified: 'UNSPECIFIED',
  Warm: 'WARM',
} as const;

export type ResourceState = ValueOf<typeof ResourceState>;

// High level Slot state
export const SlotState = {
  Free: 'FREE',
  Pending: 'PENDING',
  Potential: 'POTENTIAL',
  Running: 'RUNNING',
} as const;

export type SlotState = ValueOf<typeof SlotState>;

export const resourceStates: ResourceState[] = [
  ResourceState.Unspecified,
  ResourceState.Assigned,
  ResourceState.Pulling,
  ResourceState.Starting,
  ResourceState.Running,
  ResourceState.Warm,
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

export type SlotsRecord = { [k: string]: V1Slot };

export interface Agent {
  enabled?: boolean;
  id: string;
  registeredTime: number;
  resourcePools: string[];
  slots?: SlotsRecord;
  slotStats: Api.V1SlotStats;
  resources: Resource[];
}

export interface ClusterOverviewResource {
  /** allocated percentange of total connected slots  */
  allocation?: number;
  available: number;
  /** sum of all slots of this type across all _connected_ agents */
  total: number;
}

export type ClusterOverview = Record<ResourceType, ClusterOverviewResource>;

export type PoolOverview = Record<string, ClusterOverviewResource>;

export interface EndTimes {
  endTime?: string;
}

export interface StartEndTimes extends EndTimes {
  startTime: string;
}

/* Command */
export const CommandState = {
  Pulling: 'PULLING',
  Queued: 'QUEUED',
  Running: 'RUNNING',
  Starting: 'STARTING',
  Terminated: 'TERMINATED',
  Terminating: 'TERMINATING',
  Waiting: 'WAITING',
} as const;

export type CommandState = ValueOf<typeof CommandState>;

export type State = CommandState | typeof RunState;

export interface CommandAddress {
  containerIp: string;
  containerPort: number;
  hostIp: string;
  hostPort: number;
  protocol?: string;
}

export const CommandType = {
  Command: 'command',
  JupyterLab: 'jupyter-lab',
  Shell: 'shell',
  TensorBoard: 'tensor-board',
} as const;

export type CommandType = ValueOf<typeof CommandType>;

export interface CommandMisc {
  experimentIds: number[];
  trialIds: number[];
}

export interface CommandConfig {
  description: string;
}

// The command type is shared between Commands, JupyterLabs, TensorBoards, and Shells.
export interface Command {
  config: CommandConfig; // We do not use this field in the WebUI.
  exitStatus?: string;
  id: string;
  misc?: CommandMisc;
  registeredTime: string;
  resourcePool: string;
  serviceAddress?: string;
  state: CommandState;
  type: CommandType;
  user: User;
}

export const CheckpointStorageType = {
  AWS: 'aws',
  AZURE: 'azure',
  DIRECTORY: 'directory',
  GCS: 'gcs',
  S3: 's3',
  SharedFS: 'shared_fs',
} as const;

export type CheckpointStorageType = ValueOf<typeof CheckpointStorageType>;

export const CheckpointStorage = t.intersection([
  t.partial({
    bucket: t.string,
    containerPath: t.string,
    hostPath: t.string,
    storagePath: t.string,
    type: valueof(CheckpointStorageType),
  }),
  t.type({
    saveExperimentBest: t.number,
    saveTrialBest: t.number,
    saveTrialLatest: t.number,
  }),
]);
export type CheckpointStorage = t.TypeOf<typeof CheckpointStorage>;

export const HyperparameterType = {
  Categorical: 'categorical',
  Constant: 'const',
  Double: 'double',
  Int: 'int',
  Log: 'log',
} as const;

const HyperparametersType = valueof(HyperparameterType);
export type HyperparameterType = t.TypeOf<typeof HyperparametersType>;

export type Hyperparameters = {
  [keys: string]: Hyperparameters | HyperparameterBase;
};

const PachydermIntegrationData = t.type({
  dataset: t.type({
    branch: t.string,
    commit: t.string,
    project: t.string,
    repo: t.string,
    token: t.string,
  }),
  pachd: t.type({
    host: t.string,
    port: t.number,
  }),
  proxy: t.type({
    host: t.string,
    port: t.number,
    scheme: t.string,
  }),
});
export const Integration = t.partial({ pachyderm: PachydermIntegrationData });
export type IntegrationType = t.TypeOf<typeof Integration>;
export type PachydermIntegrationDataType = t.TypeOf<typeof PachydermIntegrationData>;
const Hyperparameters: t.RecursiveType<t.Type<Hyperparameters>> = t.recursion(
  'Hyperparameters',
  () => t.record(t.string, t.union([Hyperparameters, HyperparameterBase])),
);

// io-ts doesn't have an Omit type, so we have to build iteratively instead
const HyperparameterBaseBase = t.partial({
  base: t.number,
  count: t.number,
  maxval: t.number,
  minval: t.number,
  vals: t.array(Primitive),
});

export const HyperparameterBase = t.intersection([
  HyperparameterBaseBase,
  t.partial({
    type: HyperparametersType,
    val: t.union([Primitive, Hyperparameters]),
  }),
]);
export type HyperparameterBase = t.TypeOf<typeof HyperparameterBase>;

export const Hyperparameter = t.intersection([
  HyperparameterBaseBase,
  t.partial({
    val: Primitive,
  }),
  t.type({
    type: valueof(HyperparameterType),
  }),
]);
export type Hyperparameter = t.TypeOf<typeof Hyperparameter>;

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

export const ExperimentSearcherName = {
  AdaptiveAdvanced: 'adaptive',
  AdaptiveAsha: 'adaptive_asha',
  AdaptiveSimple: 'adaptive_simple',
  AsyncHalving: 'async_halving',
  Custom: 'custom',
  Grid: 'grid',
  Pbt: 'pbt',
  Random: 'random',
  Single: 'single',
} as const;

export type ExperimentSearcherName = ValueOf<typeof ExperimentSearcherName>;

export const ContinuableNonSingleSearcherName = new Set<ExperimentSearcherName>([
  ExperimentSearcherName.Random,
  ExperimentSearcherName.Grid,
]);

const Searcher = t.intersection([
  t.partial({
    max_length: t.record(
      t.union([t.literal('batches'), t.literal('records'), t.literal('epochs')]),
      t.number,
    ),
    max_trials: t.number,
    sourceTrialId: t.number,
  }),
  t.type({
    metric: t.string,
    name: valueof(ExperimentSearcherName),
    smallerIsBetter: t.boolean,
  }),
]);

export const ExperimentConfig = t.intersection([
  t.partial({
    checkpointStorage: CheckpointStorage,
    description: t.string,
    integrations: Integration,
    labels: t.array(t.string),
    profiling: t.type({
      enabled: t.boolean,
    }),
  }),
  t.type({
    checkpointPolicy: t.string,
    hyperparameters: Hyperparameters,
    maxRestarts: t.number,
    name: t.string,
    resources: t.partial({
      maxSlots: t.number,
    }),
    searcher: Searcher,
  }),
]);
export type ExperimentConfig = t.TypeOf<typeof ExperimentConfig>;

/* Experiment */

export const ExperimentAction = {
  Activate: 'Resume',
  Archive: 'Archive',
  Cancel: 'Stop',
  CompareTrials: 'Compare Trials',
  ContinueTrial: 'Continue Trial',
  Delete: 'Delete',
  DownloadCode: 'Download Experiment Code',
  Edit: 'Edit',
  Fork: 'Fork',
  HyperparameterSearch: 'Hyperparameter Search',
  Kill: 'Kill',
  Move: 'Move',
  OpenTensorBoard: 'View in TensorBoard',
  Pause: 'Pause',
  RetainLogs: 'Retain Logs',
  Retry: 'Retry',
  SwitchPin: 'Switch Pin',
  Unarchive: 'Unarchive',
  ViewLogs: 'View Logs',
} as const;

export type ExperimentAction = ValueOf<typeof ExperimentAction>;

export interface BulkActionResult {
  successful: number[];
  failed: Api.V1ExperimentActionResult[];
}

export interface ExperimentPagination extends WithPagination {
  experiments: BulkExperimentItem[];
}

export interface SearchExperimentPagination extends WithPagination {
  experiments: ExperimentWithTrial[];
}

export const RunState = {
  Active: 'ACTIVE',
  Canceled: 'CANCELED',
  Completed: 'COMPLETED',
  Deleted: 'DELETED',
  DeleteFailed: 'DELETE_FAILED',
  Deleting: 'DELETING',
  Error: 'ERROR',
  Paused: 'PAUSED',
  Pulling: 'PULLING',
  Queued: 'QUEUED',
  Running: 'RUNNING',
  Starting: 'STARTING',
  StoppingCanceled: 'STOPPING_CANCELED',
  StoppingCompleted: 'STOPPING_COMPLETED',
  StoppingError: 'STOPPING_ERROR',
  StoppingKilled: 'STOPPING_KILLED',
  Unspecified: 'UNSPECIFIED',
} as const;

export type RunState = ValueOf<typeof RunState>;

export interface ValidationHistory {
  endTime: string;
  trialId: number;
  validationError?: number;
}

export const CheckpointState = {
  Active: 'ACTIVE',
  Completed: 'COMPLETED',
  Deleted: 'DELETED',
  Error: 'ERROR',
  PartiallyDeleted: 'PARTIALLY_DELETED',
  Unspecified: 'UNSPECIFIED',
} as const;

export type CheckpointState = ValueOf<typeof CheckpointState>;

export const MetricType = {
  Training: 'training',
  Validation: 'validation',
} as const;

export type MetricType = ValueOf<typeof MetricType>;

export type MetricTypeParam =
  | 'METRIC_TYPE_UNSPECIFIED'
  | 'METRIC_TYPE_TRAINING'
  | 'METRIC_TYPE_VALIDATION';

export const metricTypeParamMap: Record<string, MetricTypeParam> = {
  [MetricType.Training]: 'METRIC_TYPE_TRAINING',
  [MetricType.Validation]: 'METRIC_TYPE_VALIDATION',
};

export interface Metric {
  group: string;
  name: string;
}

export interface BaseWorkload extends EndTimes {
  totalBatches: number;
}

export interface CheckpointWorkload extends BaseWorkload {
  resources?: Record<string, number>;
  state: CheckpointState;
  uuid?: string;
}

export interface CheckpointWorkloadExtended extends CheckpointWorkload {
  experimentId: number;
  trialId: number;
}

export interface MetricsWorkload extends BaseWorkload {
  metrics?: Record<string, number>;
}
export interface WorkloadGroup {
  checkpoint?: CheckpointWorkload;
  metrics: Record<string, MetricsWorkload>;
}

export const TrialWorkloadFilter = {
  All: 'All',
  Checkpoint: 'Has Checkpoint',
  CheckpointOrValidation: 'Has Checkpoint or Validation',
  Validation: 'Has Validation',
} as const;

export type TrialWorkloadFilter = ValueOf<typeof TrialWorkloadFilter>;

// This is to support the steps table in trial details and shouldn't be used
// elsewhere so we can remove it with a redesign.
export interface Step extends WorkloadGroup, StartEndTimes {
  batchNum: number;
  key: string;
  // training: MetricsWorkload;
}

type MetricStruct = Record<string, number>;
export interface Metrics extends Api.V1Metrics {
  // these two fields are present in the protos
  // as a struct and list of structs, respectively
  // here, we are being a bit more precise
  avgMetrics: MetricStruct;
  batchMetrics?: Array<MetricStruct>;
}

export type Metadata = Record<RecordKey, string | object>;

export interface CoreApiGenericCheckpoint {
  allocationId?: string;
  experimentConfig?: ExperimentConfig;
  experimentId?: number;
  hparams?: TrialHyperparameters;
  metadata: Metadata;
  reportTime?: string;
  resources: Record<string, number>;
  searcherMetric?: number;
  state: CheckpointState;
  taskId?: string;
  totalBatches: number;
  trainingMetrics?: Metrics;
  trialId?: number;
  uuid: string;
  validationMetrics?: Metrics;
}

export interface CheckpointPagination extends WithPagination {
  checkpoints: CoreApiGenericCheckpoint[];
}

export const checkpointAction = {
  Delete: 'Delete',
  Register: 'Register',
} as const;

export type CheckpointAction = ValueOf<typeof checkpointAction>;

export interface TrialPagination extends WithPagination {
  trials: TrialItem[];
}

type HpValue = Primitive | RawJson;
export type TrialHyperparameters = Record<string, HpValue>;

export interface MetricSummary {
  count?: number;
  last?: Primitive;
  max?: number;
  min?: number;
  sum?: number;
  type: 'string' | 'number' | 'boolean' | 'date' | 'object' | 'array' | 'null';
}

export interface SummaryMetrics {
  [customMetricType: string]: Record<string, MetricSummary> | null;
}

export interface TrialItem extends StartEndTimes {
  autoRestarts: number;
  bestAvailableCheckpoint?: CheckpointWorkload;
  bestValidationMetric?: MetricsWorkload;
  checkpointCount?: number;
  experimentId: number;
  hyperparameters: TrialHyperparameters;
  id: number;
  latestValidationMetric?: MetricsWorkload;
  state: RunState;
  summaryMetrics?: SummaryMetrics;
  totalBatchesProcessed: number;
  totalCheckpointSize: number;
  searcherMetricsVal?: number;
  logRetentionDays?: number;
  taskId?: string;
}

export interface TrialDetails extends TrialItem {
  runnerState?: string;
}

export interface TrialRemainingLogRetentionDays {
  remainingLogRetentionDays?: number;
}

export interface TrialWorkloads extends WithPagination {
  workloads: WorkloadGroup[];
}

export const Scale = {
  Linear: 'linear',
  Log: 'log',
} as const;

export type Scale = ValueOf<typeof Scale>;

export interface MetricDatapoint {
  batches: number;
  epoch?: number;
  time: Date;
  values: Record<string, number>;
}

export interface MetricDatapointTime {
  time: Date;
  value: number;
}

export interface MetricDatapointEpoch {
  epoch: number;
  value: number;
}

export interface MetricContainer {
  data: MetricDatapoint[];
  epochs?: MetricDatapointEpoch[];
  group: string;
  time?: MetricDatapointTime[];
}

export interface TrialSummary extends TrialItem {
  metrics: MetricContainer[];
}

// we're declaring the type here so if/when the types drift we know to fix it
export const JobSummary: t.Type<Api.V1JobSummary> = t.type({
  jobsAhead: t.number,
  state: valueof(Api.Jobv1State),
});
export type JobSummary = t.TypeOf<typeof JobSummary>;

// Bulk endpoints like experimentSearch dont return config due to perf issue
// since https://github.com/determined-ai/determined/pull/8732
export const BulkExperimentItem = t.intersection([
  t.partial({
    checkpoints: t.number,
    checkpointSize: t.number,
    config: ExperimentConfig,
    description: t.string,
    duration: t.number,
    endTime: t.string,
    externalExperimentId: t.string,
    externalTrialId: t.string,
    forkedFrom: t.number,
    jobSummary: JobSummary,
    modelDefinitionSize: t.number,
    notes: t.string,
    progress: t.number,
    projectName: t.string,
    searcherMetric: t.string,
    searcherMetricValue: t.number,
    trialIds: t.array(t.number),
    unmanaged: t.boolean,
    workspaceId: t.number,
    workspaceName: t.string,
  }),
  t.type({
    archived: t.boolean,
    hyperparameters: t.record(t.string, Hyperparameter),
    id: t.number,
    jobId: t.string,
    labels: t.array(t.string),
    name: t.string,
    numTrials: t.number,
    projectId: t.number,
    resourcePool: t.string,
    searcherType: t.string,
    startTime: t.string,
    state: t.union([valueof(RunState), valueof(Api.Jobv1State)]),
    userId: t.number,
  }),
]);
export type BulkExperimentItem = t.TypeOf<typeof BulkExperimentItem>;

export const FullExperimentItem = t.intersection([
  BulkExperimentItem,
  t.partial({
    config: ExperimentConfig,
  }),
  t.type({
    configRaw: JsonObject,
  }),
]);
export type FullExperimentItem = t.TypeOf<typeof FullExperimentItem>;

export interface ExperimentWithTrial {
  experiment: BulkExperimentItem;
  bestTrial?: TrialItem;
}

export interface ProjectExperiment extends BulkExperimentItem {
  parentArchived: boolean;
  projectName: string;
  projectOwnerId: number;
  workspaceId: number;
  workspaceName: string;
}

export interface CreateExperimentResponse {
  experiment: ExperimentBase;
  warnings?: V1LaunchWarning[];
}

export interface ExperimentBase extends ProjectExperiment {
  config: ExperimentConfig;
  configRaw: RawJson; // Readonly unparsed config object.
  hyperparameters: HyperparametersFlattened; // nested hp keys are flattened, eg) foo.bar
  originalConfig: string;
}

interface Allocation {
  isReady: boolean;
  state: CommandState;
  taskId?: string;
}

export interface TaskItem {
  allocations: Allocation[];
  taskId: string;
}

export interface TaskCounts {
  commands: number;
  notebooks: number;
  shells: number;
  tensorboards: number;
}

export interface ModelItem {
  archived?: boolean;
  creationTime: string;
  description?: string;
  id: number;
  labels?: string[];
  lastUpdatedTime: string;
  metadata: Metadata;
  name: string;
  notes?: string;
  numVersions: number;
  userId: number;
  workspaceId: number;
}

export interface ModelVersion {
  checkpoint: CoreApiGenericCheckpoint;
  comment?: string;
  creationTime: string;
  id: number;
  labels?: string[];
  lastUpdatedTime?: string;
  metadata?: Metadata;
  model: ModelItem;
  name?: string;
  notes?: string;
  userId: number;
  version: number;
}

export interface ModelPagination extends WithPagination {
  models: ModelItem[];
}

export interface ModelWithVersions extends WithPagination {
  model: ModelItem;
  modelVersions: ModelVersion[];
}

export interface Task {
  id: string;
  name: string;
  resourcePool: string;
  serviceAddress?: string;
  startTime: string;
  url?: string;
}

// CompoundRunState adds more information about a job's state to RunState.
export type CompoundRunState = RunState | JobState;

export interface ExperimentTask extends Task {
  archived: boolean;
  parentArchived: boolean;
  progress?: number;
  projectId: number;
  resourcePool: string;
  state: CompoundRunState;
  userId?: number;
  username: string;
  workspaceId: number;
}

export interface CommandResponse {
  command: CommandTask;
  warnings?: V1LaunchWarning[];
}

export interface CommandTask extends Task {
  displayName?: string;
  misc?: CommandMisc;
  resourcePool: string;
  state: CommandState;
  type: CommandType;
  userId: number;
  workspaceId: number;
}

export const TaskAction = {
  Connect: 'Connect',
  Kill: 'Kill',
  ViewLogs: 'View Logs',
} as const;

export type TaskAction = ValueOf<typeof TaskAction>;

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

export const TaskType = {
  Command: 'command',
  Experiment: 'experiment',
  JupyterLab: 'jupyter-lab',
  Shell: 'shell',
  TensorBoard: 'tensor-board',
} as const;

export type TaskType = ValueOf<typeof TaskType>;

export const ArchiveFilter = {
  Archived: 'archived',
  Unarchived: 'unarchived',
} as const;

export type ArchiveFilter = ValueOf<typeof ArchiveFilter>;

export interface ExperimentFilters {
  archived?: ArchiveFilter;
  labels?: string[];
  states?: string[];
  users?: string[];
}

export interface ExperimentTrialFilters {
  states?: string[];
}

export interface TaskFilters<T extends CommandType | TaskType = TaskType> {
  limit: number;
  states?: string[];
  types?: T[];
  users?: string[];
  workspaces?: string[];
}

export const LogLevel = {
  Critical: 'critical',
  Debug: 'debug',
  Error: 'error',
  Info: 'info',
  None: 'none',
  Trace: 'trace',
  Warning: 'warning',
} as const;

export type LogLevel = ValueOf<typeof LogLevel>;

// Disable `sort-keys` to sort LogLevel by higher severity levels
export const LogLevelFromApi = {
  Critical: 'LOG_LEVEL_CRITICAL',
  Error: 'LOG_LEVEL_ERROR',
  Warning: 'LOG_LEVEL_WARNING',
  // eslint-disable-next-line sort-keys-fix/sort-keys-fix
  Info: 'LOG_LEVEL_INFO',
  // eslint-disable-next-line sort-keys-fix/sort-keys-fix
  Debug: 'LOG_LEVEL_DEBUG',
  Trace: 'LOG_LEVEL_TRACE',
  Unspecified: 'LOG_LEVEL_UNSPECIFIED',
} as const;

export type LogLevelFromApi = ValueOf<typeof LogLevelFromApi>;

export interface Log {
  id: number | string;
  level?: LogLevel;
  message: string;
  meta?: string;
  time: string;
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
  workspaceId: number;
}
export interface KubernetesResourceManagers {
  names: string[];
}

export interface ResourcePool extends Omit<Api.V1ResourcePool, 'slotType'> {
  slotType: ResourceType;
}

/* Jobs */

export interface LimitedJob extends Api.V1LimitedJob {
  summary: Api.V1JobSummary;
}
export interface FullJob extends Api.V1Job {
  summary: Api.V1JobSummary;
}
export type Job = LimitedJob | FullJob;
export const JobType = Api.Jobv1Type;
export type JobType = Api.Jobv1Type;
export const JobState = Api.Jobv1State;
export type JobState = Api.Jobv1State;

export const JobAction = {
  Cancel: 'Cancel',
  Kill: 'Kill',
  ManageJob: 'Manage Job',
  MoveToTop: 'Move To Top',
  ViewLog: 'View Logs',
} as const;

export type JobAction = ValueOf<typeof JobAction>;

/* End of Jobs */

export interface Workspace {
  agentUserGroup?: V1AgentUserGroup;
  archived: boolean;
  // eslint-disable-next-line  @typescript-eslint/no-explicit-any
  checkpointStorageConfig?: any;
  id: number;
  immutable: boolean;
  name: string;
  numExperiments: number;
  numProjects: number;
  pinned: boolean;
  pinnedAt?: Date;
  state: WorkspaceState;
  userId: number;
  defaultComputePool?: string;
  defaultAuxPool?: string;
}

export interface WorkspaceNamespaceBindings {
  namespaceBindings: Record<string, Api.V1WorkspaceNamespaceBinding>;
}

export interface WorkspaceResourceQuotas {
  resourceQuotas: Record<string, number>;
}

export interface WorkspacePagination extends WithPagination {
  workspaces: Workspace[];
}

export interface DeletionStatus {
  completed: boolean;
}

export const WorkspaceState = {
  Deleted: 'DELETED',
  DeleteFailed: 'DELETE_FAILED',
  Deleting: 'DELETING',
  Unspecified: 'UNSPECIFIED',
} as const;

export type WorkspaceState = ValueOf<typeof WorkspaceState>;

export const Note = t.type({
  contents: t.string,
  name: t.string,
});
export type Note = t.TypeOf<typeof Note>;

export const Project = t.intersection([
  t.type({
    archived: t.boolean,
    id: t.number,
    immutable: t.boolean,
    name: t.string,
    notes: t.array(Note),
    state: valueof(WorkspaceState),
    userId: t.number,
    workspaceId: t.number,
  }),
  t.partial({
    description: t.string,
    lastExperimentStartedAt: t.string,
    numActiveExperiments: t.number,
    numExperiments: t.number,
    numRuns: t.number,
    workspaceName: t.string,
  }),
]);

export type Project = t.TypeOf<typeof Project>;

export interface ProjectPagination extends WithPagination {
  projects: Project[];
}

export interface ProjectColumn {
  column: string;
  location: Api.V1LocationType;
  type: Api.V1ColumnType;
  displayName?: string;
}

export interface ProjectMetricsRange {
  metricsName: string;
  min: number;
  max: number;
}

export interface Permission {
  id: Api.V1PermissionType;
  scopeCluster: boolean;
  scopeWorkspace: boolean;
}

export interface UserRole {
  fromUser?: boolean;
  id: number;
  name: string;
  permissions: Permission[];
  scopeCluster?: boolean;
}

export interface UserAssignment {
  roleId: number;
  scopeCluster: boolean;
  workspaces?: number | number[];
}

export interface PermissionsSummary {
  assignments: UserAssignment[];
  roles: UserRole[];
}

export interface ExperimentPermissionsArgs {
  experiment: ProjectExperiment;
}

export interface FlatRunPermissionsArgs {
  flatRun: FlatRun;
}

export interface PermissionWorkspace {
  id: number;
  userId?: number;
}

export interface WorkspacePermissionsArgs {
  workspace?: PermissionWorkspace;
}

export interface WorkspaceMembersResponse {
  assignments: Api.V1RoleWithAssignments[];
  groups: Api.V1Group[];
  usersAssignedDirectly: DetailedUser[];
}

export interface Webhook {
  id: number;
  triggers: V1Trigger[];
  url: string;
  webhookType: string;
}

export type UserOrGroup = User | V1Group;

export type GroupWithRoleInfo = {
  groupId: Api.V1Group['groupId'];
  groupName: Api.V1Group['name'];
  roleAssignment: Api.V1RoleAssignment;
};

export type UserWithRoleInfo = {
  displayName: User['displayName'];
  roleAssignment: Api.V1RoleAssignment;
  userId: User['id'];
  username: User['username'];
};

export type UserOrGroupWithRoleInfo = UserWithRoleInfo | GroupWithRoleInfo;

export interface HpTrialData {
  data: Record<string, Primitive[]>;
  metricRange?: Range<number>;
  metricValues: number[];
  recordIds: number[];
}

/**
 * @typedef Serie
 * Represents a single Series to display on the chart.
 * @param {string} [color] - A CSS-compatible color to directly set the line and tooltip color for the Serie. Defaults to glasbeyColor.
 * @param {Partial<Record<XAxisDomain, [x: number, y: number][]>>} data - An array of ordered [x, y] points for each axis.
 * @param {string} [name] - Name to display in legend and toolip instead of Series number.
 */

export interface Serie {
  color?: string;
  data: Partial<Record<XAxisDomain, [x: number, y: number][]>>;
  key?: number;
  name?: string;
}

export const XAxisDomain = {
  Batches: 'Batches',
  Epochs: 'Epoch',
  Time: 'Time',
} as const;

export type XAxisDomain = ValueOf<typeof XAxisDomain>;

export interface FlatRun {
  id: number;
  startTime: Date | DateString;
  endTime?: Date | DateString;
  state: RunState;
  labels?: Array<string>;
  checkpointSize: number;
  checkpointCount: number;
  searcherMetricValue?: number;
  externalRunId?: number;
  hyperparameters?: TrialHyperparameters;
  summaryMetrics?: SummaryMetrics;
  userId?: number;
  duration?: number;
  projectId: number;
  projectName: string;
  workspaceId: number;
  workspaceName: string;
  archived: boolean;
  parentArchived: boolean;
  experiment?: FlatRunExperiment;
}

export interface FlatRunExperiment {
  id: number;
  searcherType: string;
  searcherMetric: string;
  forkedFrom?: number;
  externalExperimentId?: string;
  resourcePool: string;
  progress: number;
  description: string;
  name: string;
  unmanaged: boolean;
  isMultitrial: boolean;
}
export interface SearchFlatRunPagination extends WithPagination {
  runs: FlatRun[];
}

export const FlatRunAction = {
  Archive: 'Archive',
  Delete: 'Delete',
  Kill: 'Kill',
  Move: 'Move',
  Pause: 'Pause',
  Resume: 'Resume',
  Unarchive: 'Unarchive',
} as const;

export type FlatRunAction = ValueOf<typeof FlatRunAction>;

export const SelectAllType = t.type({
  exclusions: t.array(t.number),
  type: t.literal('ALL_EXCEPT'),
});

export const RegularSelectionType = t.type({
  selections: t.array(t.number),
  type: t.literal('ONLY_IN'),
});

export const SelectionType = t.union([RegularSelectionType, SelectAllType]);
export type SelectionType = t.TypeOf<typeof SelectionType>;
