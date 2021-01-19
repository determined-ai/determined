import { ColumnType } from 'antd/es/table';

import { ResourcePool } from 'types';
import { alphanumericSorter, numericSorter } from 'utils/data';

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
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      alphanumericSorter(a.description, b.description),
    title: 'Description',
  },
  {
    key: 'chart',
    title: 'GPU Slots Allocation',
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
