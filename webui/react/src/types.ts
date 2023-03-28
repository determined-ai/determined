import * as Api from 'services/api-ts-sdk';
import { V1AgentUserGroup, V1Group, V1LaunchWarning, V1Trigger } from 'services/api-ts-sdk';
import { Primitive, RawJson, RecordKey, ValueOf } from 'shared/types';

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
  agentUserGroup?: V1AgentUserGroup;
  id: number;
  isActive: boolean;
  isAdmin: boolean;
}

export interface DetailedUserList extends WithPagination {
  users: DetailedUser[];
}

export interface Auth {
  isAuthenticated: boolean;
  token?: string;
}

export interface SsoProvider {
  name: string;
  ssoUrl: string;
}

export const BrandingType = {
  Determined: 'determined',
  HPE: 'hpe',
} as const;

export type BrandingType = ValueOf<typeof BrandingType>;

export interface DeterminedInfo {
  branding?: BrandingType;
  checked: boolean;
  clusterId: string;
  clusterName: string;
  externalLoginUri?: string;
  externalLogoutUri?: string;
  featureSwitches: string[];
  isTelemetryEnabled: boolean;
  masterId: string;
  rbacEnabled: boolean;
  ssoProviders?: SsoProvider[];
  userManagementEnabled: boolean;
  version: string;
}

export interface Telemetry {
  enabled: boolean;
  segmentKey?: string;
}

export const ResourceType = {
  ALL: 'ALL',
  CPU: 'CPU',
  CUDA: 'CUDA',
  ROCM: 'ROCM',
  UNSPECIFIED: 'UNSPECIFIED',
} as const;

export type ResourceType = ValueOf<typeof ResourceType>;

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
  hostIp?: string;
  hostPort?: number;
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
  GCS: 'gcs',
  HDFS: 'hdfs',
  S3: 's3',
  SharedFS: 'shared_fs',
} as const;

export type CheckpointStorageType = ValueOf<typeof CheckpointStorageType>;

interface CheckpointStorage {
  bucket?: string;
  hostPath?: string;
  saveExperimentBest: number;
  saveTrialBest: number;
  saveTrialLatest: number;
  storagePath?: string;
  type?: CheckpointStorageType;
}

export type HpImportance = Record<string, number>;
export type HpImportanceMetricMap = Record<string, HpImportance>;
export type HpImportanceMap = { [key in MetricType]: HpImportanceMetricMap };

export const HyperparameterType = {
  Categorical: 'categorical',
  Constant: 'const',
  Double: 'double',
  Int: 'int',
  Log: 'log',
} as const;

export type HyperparameterType = ValueOf<typeof HyperparameterType>;

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

export interface ExperimentConfig {
  checkpointPolicy: string;
  checkpointStorage?: CheckpointStorage;
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
    max_length?: Record<'batches' | 'records' | 'epochs', number>;
    max_trials?: number;
    metric: string;
    name: ExperimentSearcherName;
    smallerIsBetter: boolean;
    sourceTrialId?: number;
  };
}

/* Experiment */

export const ExperimentAction = {
  Activate: 'Activate',
  Archive: 'Archive',
  Cancel: 'Cancel',
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
  SwitchPin: 'Switch Pin',
  Unarchive: 'Unarchive',
  ViewLogs: 'View Logs',
} as const;

export type ExperimentAction = ValueOf<typeof ExperimentAction>;

export interface ExperimentPagination extends WithPagination {
  experiments: ExperimentItem[];
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
  totalBatchesProcessed: number;
  totalCheckpointSize: number;
}

export interface TrialDetails extends TrialItem {
  runnerState?: string;
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
  value: number;
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
  name: string;
  time?: MetricDatapointTime[];
  type: MetricType;
}

export interface TrialSummary extends TrialItem {
  metrics: MetricContainer[];
}

export interface ExperimentItem {
  archived: boolean;
  checkpointCount?: number;
  checkpointSize?: number;
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
  projectName?: string;
  resourcePool: string;
  searcherMetricValue?: number;
  searcherType: string;
  startTime: string;
  state: CompoundRunState;
  trialIds?: number[];
  userId: number;
  workspaceId?: number;
  workspaceName?: string;
}

export interface ProjectExperiment extends ExperimentItem {
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

export interface ModelVersions extends WithPagination {
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
}

export interface ResourcePool extends Omit<Api.V1ResourcePool, 'slotType'> {
  slotType: ResourceType;
}

/* Jobs */

export interface Job extends Api.V1Job {
  summary: Api.V1JobSummary;
}
export const JobType = Api.Jobv1Type;
export type JobType = Api.Jobv1Type;
export const JobState = Api.Jobv1State;
export type JobState = Api.Jobv1State;
export type JobSummary = Api.V1JobSummary;
export type RPStats = Api.V1RPQueueStat;

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
  usersAssignedDirectly: User[];
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
