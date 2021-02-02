import { ColumnType } from 'antd/es/table';

import { ResourcePool } from 'types';
import { alphanumericSorter, numericSorter } from 'utils/data';
import { V1ResourcePoolTypeToLabel } from 'utils/types';

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
    render: (_, record) => V1ResourcePoolTypeToLabel[record.type],
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
    dataIndex: 'numSlots',
    key: 'numSlots',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.numSlots, b.numSlots),
    title: 'Total Slots',
  },
  {
    dataIndex: 'cpuContainerCapacity',
    key: 'cpuContainerCapacity',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.cpuContainerCapacity, b.cpuContainerCapacity),
    title: 'Max CPU Containers Per Agent',
  },
  {
    dataIndex: 'cpuContainersRunning',
    key: 'cpuContainersRunning',
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      numericSorter(a.cpuContainersRunning, b.cpuContainersRunning),
    title: 'CPUs Used',
  },
];
