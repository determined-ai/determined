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

export type State = CommandState | RunState;

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
  state: CommandState;
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
  state: RunState;
}

export interface Task {
  title: string;
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
  username?: string;
}

export type RecentEvent = {
  lastEvent: {
    name: string;
    date: string;
  };
};

export type AnyTask = CommandTask | ExperimentTask;
export type RecentTask = AnyTask & RecentEvent;
export type RecentCommandTask = CommandTask & RecentEvent;
export type RecentExperimentTask = ExperimentTask & RecentEvent;

export type PropsWithClassName<T> = T & {className?: string};

export type TaskType = CommandType | 'Experiment';

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

export const terminalCommandStates: Set<CommandState> = new Set([
  CommandState.Terminated,
]);

export const terminalRunStates: Set<RunState> = new Set([
  RunState.Errored,
  RunState.Canceled,
]);
