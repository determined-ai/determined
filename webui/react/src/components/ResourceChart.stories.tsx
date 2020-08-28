import React from 'react';

import { Resource, ResourceState } from 'types';

import ResourceChart from './ResourceChart';

export default {
  component: ResourceChart,
  title: 'ResourceChart',
};

export const Default = (): React.ReactNode => <ResourceChart
  resources={[
    { container: undefined },
    { container: { state: ResourceState.Assigned } },
    { container: { state: ResourceState.Pulling } },
    { container: { state: ResourceState.Pulling } },
    { container: { state: ResourceState.Pulling } },
    { container: { state: ResourceState.Terminated } },
    { container: { state: ResourceState.Terminated } },
    { container: { state: ResourceState.Terminated } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
  ] as Resource[]}
  title="TPUs" />;

export const AllSame = (): React.ReactNode => <ResourceChart
  resources={[
    { container: undefined },
    { container: undefined },
    { container: undefined },
    { container: undefined },
    { container: undefined },
    { container: undefined },
    { container: undefined },
  ] as Resource[]}
  title="TPUs" />;

export const HalfHalf = (): React.ReactNode => <ResourceChart
  resources={[
    { container: undefined },
    { container: undefined },
    { container: undefined },
    { container: undefined },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
  ] as Resource[]}
  title="GPUs" />;

export const MostlyStarting = (): React.ReactNode => <ResourceChart
  resources={[
    { container: undefined },
    { container: undefined },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Starting } },
    { container: { state: ResourceState.Starting } },
    { container: { state: ResourceState.Starting } },
    { container: { state: ResourceState.Starting } },
  ] as Resource[]}
  title="GPUs" />;

export const MostlyRunning = (): React.ReactNode => <ResourceChart
  resources={[
    { container: undefined },
    { container: undefined },
    { container: { state: ResourceState.Starting } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
    { container: { state: ResourceState.Running } },
  ] as Resource[]}
  title="GPUs" />;

export const Empty = (): React.ReactNode => <ResourceChart resources={[]} title="DPUs" />;
