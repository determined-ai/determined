import { ColumnType } from 'antd/es/table';
import React from 'react';

import SlotAllocationBar from 'components/SlotAllocationBar';
import { ResourceState } from 'types';
import { ResourcePool } from 'types/ResourcePool';
import { alphanumericSorter, numericSorter } from 'utils/data';

import css from './HGICluster.table.module.scss';

const descriptionRender = (_: unknown, record: ResourcePool): React.ReactNode =>
  <div className={css.descriptionColumn}>{record.description}</div>;

const chartRender = (_:unknown, record: ResourcePool): React.ReactNode => {
  const containerStates: ResourceState[] = [
    ResourceState.Assigned, ResourceState.Pulling, ResourceState.Running,
  ]; // GPU. TODO this would come from Agent data.

  return <div className={css.chartColumn}>
    <SlotAllocationBar
      className={css.chartColumn}
      hideHeader
      resourceStates={containerStates}
      totalSlots={record.numAgents * record.gpusPerAgent} />
  </div>;

};

export const columns: ColumnType<ResourcePool>[] = [
  {
    dataIndex: 'name',
    key: 'name',
    sorter: (a: ResourcePool, b: ResourcePool): number => alphanumericSorter(a.name, b.name),
    title: 'Pool Name',
  },
  {
    dataIndex: 'description',
    key: 'description',
    render: descriptionRender,
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      alphanumericSorter(a.description, b.description),
    title: 'Description',
  },
  {
    key: 'chart',
    render: chartRender,
    title: 'Chart',
  },
  {
    dataIndex: 'type',
    key: 'type',
    sorter: (a: ResourcePool, b: ResourcePool): number => alphanumericSorter(a.type, b.type),
    title: 'Type',
  },
  {
    dataIndex: 'numAgents',
    key: 'numAgents',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.numAgents, b.numAgents),
    title: 'Agents',
  },
  {
    dataIndex: 'slotsAvailable',
    key: 'slotsAvailable',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.slotsAvailable, b.slotsAvailable),
    title: 'Slots Available',
  },
  {
    dataIndex: 'cpuContainerCapacity',
    key: 'cpuContainerCapacity',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.cpuContainerCapacity, b.cpuContainerCapacity),
    title: 'CPUs Available',
  },
  {
    dataIndex: 'cpuContainersRunning',
    key: 'cpuContainersRunning',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.cpuContainersRunning, b.cpuContainersRunning),
    title: 'CPUs Used',
  },
];
