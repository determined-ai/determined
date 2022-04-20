import React, { useCallback, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { V1ResourcePoolTypeToLabel, V1SchedulerTypeToLabel } from 'constants/states';
import { useStore } from 'contexts/Store';
import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import awsLogo from 'shared/assets/images/aws-logo.svg';
import gcpLogo from 'shared/assets/images/gcp-logo.svg';
import k8sLogo from 'shared/assets/images/k8s-logo.svg';
import staticLogo from 'shared/assets/images/on-prem-logo.svg';
import { clone } from 'shared/utils/data';
import { ShirtSize } from 'themes';
import { ResourcePool, ResourceState, ResourceType } from 'types';

import Json from './Json';
import Link from './Link';
import css from './ResourcePoolCardLight.module.scss';
import ResourcePoolDetails from './ResourcePoolDetails';

interface Props {
  computeContainerStates: ResourceState[];
  resourcePool: ResourcePool;
  resourceType: ResourceType;
  totalComputeSlots: number;
}

export const poolLogo = (type: V1ResourcePoolType): React.ReactNode => {
  let iconSrc = '';
  switch (type) {
    case V1ResourcePoolType.AWS:
      iconSrc = awsLogo;
      break;
    case V1ResourcePoolType.GCP:
      iconSrc = gcpLogo;
      break;
    case V1ResourcePoolType.K8S:
      iconSrc = k8sLogo;
      break;
    case V1ResourcePoolType.UNSPECIFIED:
    case V1ResourcePoolType.STATIC:
      iconSrc = staticLogo;
      break;
  }

  return <img src={iconSrc} />;
};

const poolAttributes = [
  {
    key: 'accelerator',
    label: 'Accelerator',
    render: (x: ResourcePool) => x.accelerator ? x.accelerator : '--',
  },
  {
    key: 'type',
    label: 'Type',
    render: (x: ResourcePool) => V1ResourcePoolTypeToLabel[x.type],
  },
  { key: 'instanceType', label: 'Instance Type' },
  {
    key: 'numAgents',
    label: 'Connected Agents',
    render: (x: ResourcePool) => {
      if (x.maxAgents > 0) {
        return `${x.numAgents}/${x.maxAgents}`;
      }
      return x.numAgents;
    },
  },
  { key: 'slotsPerAgent', label: 'Slots Per Agent' },
  { key: 'auxContainerCapacityPerAgent', label: 'Aux Containers Per Agent' },
];

type SafeRawJson = Record<string, unknown>;

const ResourcePoolCardLight: React.FC<Props> = ({
  computeContainerStates,
  resourcePool: pool,
  totalComputeSlots,
}: Props) => {
  const [ detailVisible, setDetailVisible ] = useState(false);

  const { pool: poolOverview } = useStore();

  const descriptionClasses = [ css.description ];

  if (!pool.description) descriptionClasses.push(css.empty);

  const isAux = useMemo(() => {
    return pool.auxContainerCapacityPerAgent > 0;
  }, [ pool ]);

  const processedPool = useMemo(() => {
    const newPool = clone(pool);
    Object.keys(newPool).forEach(key => {
      const value = pool[key as keyof ResourcePool];
      if (key === 'slotsPerAgent' && value === -1) newPool[key] = 'Unknown';
      if (key === 'schedulerType') newPool[key] = V1SchedulerTypeToLabel[value as V1SchedulerType];
    });
    return newPool;
  }, [ pool ]);

  const shortDetails = useMemo(() => {
    return poolAttributes.reduce((acc, attribute) => {
      const value = attribute.render ?
        attribute.render(processedPool) :
        processedPool[attribute.key as keyof ResourcePool];
      acc[attribute.label] = value;
      if (!isAux && attribute.key === 'auxContainerCapacityPerAgent') delete acc[attribute.label];
      if (pool.type === V1ResourcePoolType.K8S && attribute.key !== 'type') {
        delete acc[attribute.label];
      }
      return acc;
    }, {} as SafeRawJson);
  }, [ processedPool, isAux, pool ]);

  const toggleModal = useCallback(() => setDetailVisible((cur: boolean) => !cur), []);

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.info}>
          <div className={css.name}>{pool.name}</div>
        </div>
        <div className={css.default}>
          {(pool.defaultAuxPool || pool.defaultComputePool) && <span>default</span>}
          <Link onClick={toggleModal}><Icon name="info" /></Link>
        </div>
      </div>
      <div className={css.body}>
        <section>
          {poolOverview[pool.name]?.total > 0 && (
            <SlotAllocationBar
              footer={{ queued: pool?.stats?.queuedCount }}
              hideHeader
              poolType={pool.type}
              resourceStates={computeContainerStates}
              size={ShirtSize.medium}
              slotsPotential={totalComputeSlots}
              totalSlots={pool.slotsAvailable}
            />
          )}
          {isAux && (
            <SlotAllocationBar
              footer={{
                auxContainerCapacity: pool.auxContainerCapacity,
                auxContainersRunning: pool.auxContainersRunning,
              }}
              hideHeader
              isAux={true}
              poolType={pool.type}
              resourceStates={computeContainerStates}
              size={ShirtSize.medium}
              totalSlots={totalComputeSlots}
            />
          )}
        </section>
        <section className={css.details}>
          <Json hideDivider json={shortDetails} />
          <div>
            <ResourcePoolDetails
              finally={toggleModal}
              resourcePool={processedPool}
              visible={detailVisible}
            />
          </div>
        </section>
        <div />
      </div>
    </div>
  );
};

export default ResourcePoolCardLight;
