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

export interface CommandAddress {
  containerIp: string;
  containerPort: number;
  hostIp: string;
  hostPort: number;
  protocol: string;
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
  privateKey?: string;
}

export interface CommandConfig {
  description: string;
}

// The command type is shared between Commands, Notebooks, Tensorboards, and Shells.
export interface Command {
  addresses?: CommandAddress[];
  kind: CommandType;
  config: CommandConfig; // We do not use this field in the WebUI.
  exitStatus?: string;
  id: string;
  misc?: CommandMisc;
  owner: Owner;
  registeredTime: string;
  serviceAddress?: string;
  state: string;
}

// TODO compelete the config object as we start using different attributes.
export interface ExperimentConfig {
  description: string;
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

export interface Experiment {
  archived: boolean;
  config: ExperimentConfig;
  endTime?: string;
  id: number;
  ownerId: number;
  progress?: number;
  startTime: string;
  state: string;
}

export enum TaskType {
  Command = 'COMMAND',
  Experiment = 'EXPERIMENT',
  Notebook = 'NOTEBOOK',
  Shell = 'SHELL',
  Tensorboard = 'TENSORBOARD',
}

export interface RecentTask {
  archived?: boolean;
  title: string;
  type: TaskType;
  lastEvent: {
    name: string;
    date: string;
  };
  id: string;
  ownerId: number;
  progress?: number;
  url?: string;
  state: RunState | CommandState;
}

export type PropsWithClassName<T> = T & {className?: string};

export enum TBSourceType {
  Trial,
  Experiment
}

export type CommonProps = {
  className?: string;
  children?: React.ReactNode;
}
