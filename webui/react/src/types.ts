import * as Api from 'services/api-ts-sdk';
import { Primitive, RawJson, RecordKey } from 'shared/types';

interface WithPagination {
  pagination: Api.V1Pagination; // probably should use this or Pagination
}

export type PropsWithStoragePath<T> = T & { storagePath?: string };

export interface User {
  displayName?: string;
  id: number;
  modifiedAt?: number;
  username: string;
}

export interface DetailedUser extends User {
  id: number;
  isActive: boolean;
  isAdmin: boolean;
}

export interface DetailedUserList extends WithPagination {
  users: DetailedUser[],
}

export interface Auth {
  isAuthenticated: boolean;
  token?: string;
  user?: DetailedUser;
}

export interface SsoProvider {
  name: string;
  ssoUrl: string;
}

export enum BrandingType {
  Determined = 'determined',
  HPE = 'hpe',
}

export interface DeterminedInfo {
  branding?: BrandingType;
  checked: boolean,
  clusterId: string;
  clusterName: string;
  externalLoginUri?: string;
  externalLogoutUri?: string;
  isTelemetryEnabled: boolean;
  masterId: string;
  ssoProviders?: SsoProvider[];
  version: string;
}

export interface Telemetry {
  enabled: boolean;
  segmentKey?: string;
}

export enum ResourceType {
  CPU = 'CPU',
  CUDA = 'CUDA',
  ROCM = 'ROCM',
  ALL = 'ALL',
  UNSPECIFIED = 'UNSPECIFIED',
}

export const deviceTypes = new Set([ ResourceType.CPU, ResourceType.CUDA, ResourceType.ROCM ]);

export enum ResourceState { // This is almost CommandState
  Unspecified = 'UNSPECIFIED',
  Assigned = 'ASSIGNED',
  Pulling = 'PULLING',
  Starting = 'STARTING',
  Running = 'RUNNING',
  Terminated = 'TERMINATED',
  Warm = 'WARM',
  Potential = 'POTENTIAL'
}

// High level Slot state
export enum SlotState {
  Running = 'RUNNING',
  Free = 'FREE',
  Pending = 'PENDING',
  Potential = 'POTENTIAL'
}

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

export interface Agent {
  id: string;
  registeredTime: number;
  resourcePools: string[];
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
  Command = 'command',
  JupyterLab = 'jupyter-lab',
  Shell = 'shell',
  TensorBoard = 'tensor-board',
}

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
  AsyncHalving = 'async_halving',
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
  maxRestarts: number;
  name: string;
  profiling?: {
    enabled: boolean;
  };
  resources: {
    maxSlots?: number;
  };
  searcher: {
    max_length?: Record<'batches' | 'records' | 'epochs', number>,
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
  CompareExperiments = 'Compare',
  CompareTrials = 'Compare Trials',
  ContinueTrial = 'Continue Trial',
  Delete = 'Delete',
  DownloadCode = 'Download Experiment Code',
  Fork = 'Fork',
  HyperparameterSearch = 'Hyperparameter Search',
  Kill = 'Kill',
  Move = 'Move',
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
  training?: MetricsWorkload;
  validation?: MetricsWorkload;
}

export enum TrialWorkloadFilter {
  All = 'All',
  Checkpoint = 'Has Checkpoint',
  Validation = 'Has Validation',
  CheckpointOrValidation = 'Has Checkpoint or Validation',
}

// This is to support the steps table in trial details and shouldn't be used
// elsewhere so we can remove it with a redesign.
export interface Step extends WorkloadGroup, StartEndTimes {
  batchNum: number;
  key: string;
  training: MetricsWorkload;
}

type MetricStruct = Record<string, number>;
export interface Metrics extends Api.V1Metrics {
  // these two fields are present in the protos
  // as a struct and list of structs, respectively
  // here, we are being a bit more precise
  avgMetrics: MetricStruct;
  batchMetrics?: Array<MetricStruct>;
}

export type Metadata = Record<RecordKey, string>;

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

export interface TrialPagination extends WithPagination {
  trials: TrialItem[];
}

type HpValue = Primitive | RawJson
export type TrialHyperparameters = Record<string, HpValue>

export interface TrialItem extends StartEndTimes {
  autoRestarts: number;
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
  totalCheckpointSize: number;
}

export interface TrialWorkloads extends WithPagination {
  workloads: WorkloadGroup[];
}

export enum Scale {
  Linear = 'linear',
  Log = 'log',
}

export interface MetricDatapoint {
  batches: number;
  value: number;
}

export interface MetricContainer {
  data: MetricDatapoint[];
  name: string;
  type: MetricType;
}

export interface TrialSummary extends TrialItem {
  metrics: MetricContainer[];
}

export interface ExperimentItem {
  archived: boolean;
  config: ExperimentConfig;
  configRaw: RawJson; // Readonly unparsed config object.
  description?: string;
  endTime?: string;
  forkedFrom?: number;
  hyperparameters: HyperparametersFlattened; // Nested HP keys are flattened, eg) foo.bar
  id: number;
  jobId: string;
  jobSummary?: JobSummary;
  labels: string[];
  name: string;
  notes?: string;
  numTrials: number;
  progress?: number;
  projectId: number;
  resourcePool: string;
  searcherType: string;
  startTime: string;
  state: CompoundRunState;
  trialIds?: number[];
  userId: number;
}

export interface ProjectExperiment extends ExperimentItem {
  parentArchived: boolean;
  projectName: string;
  projectOwnerId: number;
  workspaceId: number;
  workspaceName: string;
}

export interface ExperimentBase extends ProjectExperiment {
  config: ExperimentConfig;
  configRaw: RawJson; // Readonly unparsed config object.
  hyperparameters: HyperparametersFlattened; // nested hp keys are flattened, eg) foo.bar
  originalConfig: string;
}

// TODO we should be able to remove ExperimentOld but leaving this off.
export interface ExperimentOld extends ExperimentItem {
  config: ExperimentConfig;
  configRaw: RawJson; // Readonly unparsed config object.
  hyperparameters: HyperparametersFlattened; // nested hp keys are flattened, eg) foo.bar
  url: string;
}

export enum ExperimentVisualizationType {
  HpParallelCoordinates = 'hp-parallel-coordinates',
  HpHeatMap = 'hp-heat-map',
  HpScatterPlots = 'hp-scatter-plots',
  LearningCurve = 'learning-curve',
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

export interface ModelVersions extends WithPagination {
  model: ModelItem;
  modelVersions: ModelVersion[]
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
export type CompoundRunState = RunState | JobState

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

export interface CommandTask extends Task {
  displayName?: string;
  misc?: CommandMisc;
  resourcePool: string;
  state: CommandState;
  type: CommandType;
  userId: number;
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

export enum TaskType {
  Command = 'command',
  Experiment = 'experiment',
  JupyterLab = 'jupyter-lab',
  Shell = 'shell',
  TensorBoard = 'tensor-board',
}

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

export interface TaskFilters<T extends CommandType | TaskType = TaskType> {
  limit: number;
  states?: string[];
  types?: T[];
  users?: string[];
}

export enum LogLevel {
  Critical = 'critical',
  Debug = 'debug',
  Error = 'error',
  Info = 'info',
  None = 'none',
  Trace = 'trace',
  Warning = 'warning',
}

export enum LogLevelFromApi {
  Unspecified = 'LOG_LEVEL_UNSPECIFIED',
  Trace = 'LOG_LEVEL_TRACE',
  Debug = 'LOG_LEVEL_DEBUG',
  Info = 'LOG_LEVEL_INFO',
  Warning = 'LOG_LEVEL_WARNING',
  Error = 'LOG_LEVEL_ERROR',
  Critical = 'LOG_LEVEL_CRITICAL',
}

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
}

export interface ResourcePool extends Omit<Api.V1ResourcePool, 'slotType'> {
  slotType: ResourceType;
}

/* Jobs */

export interface Job extends Api.V1Job {
  summary: Api.V1JobSummary;
}
export const JobType = Api.Determinedjobv1Type;
export type JobType = Api.Determinedjobv1Type;
export const JobState = Api.Determinedjobv1State;
export type JobState = Api.Determinedjobv1State;
export type JobSummary = Api.V1JobSummary;
export type RPStats = Api.V1RPQueueStat;

export enum JobAction {
  Cancel = 'Cancel',
  Kill = 'Kill',
  ManageJob = 'Manage Job',
  MoveToTop = 'Move To Top',
  ViewLog = 'View Logs',
}

/* End of Jobs */

export interface Workspace {
  archived: boolean;
  id: number;
  immutable: boolean;
  name: string;
  numExperiments: number;
  numProjects: number;
  pinned: boolean;
  state: WorkspaceState;
  userId: number;
}

export interface WorkspacePagination extends WithPagination {
  workspaces: Workspace[];
}

export interface DeletionStatus {
  completed: boolean;
}

export enum WorkspaceState {
  Deleted = 'DELETED',
  DeleteFailed = 'DELETE_FAILED',
  Deleting = 'DELETING',
  Unspecified = 'UNSPECIFIED',
}

export interface Note {
  contents: string;
  name: string;
}
export interface Project {
  archived: boolean;
  description?: string;
  id: number;
  immutable: boolean;
  lastExperimentStartedAt?: Date;
  name: string;
  notes: Note[];
  numActiveExperiments: number;
  numExperiments: number;
  state: WorkspaceState;
  userId: number;
  workspaceId: number;
  workspaceName: string;
}

export interface ProjectPagination extends WithPagination {
  projects: Project[];
}
