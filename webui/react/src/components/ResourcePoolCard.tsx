import React, { useCallback, useMemo, useState } from 'react';

import awsLogo from 'assets/images/aws-logo.svg';
import gcpLogo from 'assets/images/gcp-logo.svg';
import k8sLogo from 'assets/images/k8s-logo.svg';
import staticLogo from 'assets/images/on-prem-logo.svg';
import Badge, { BadgeType } from 'components/Badge';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { V1ResourcePoolTypeToLabel, V1SchedulerTypeToLabel } from 'constants/states';
import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import { deviceTypes, ResourcePool, ResourceState, ResourceType } from 'types';
import { clone } from 'utils/data';

import Json from './Json';
import Link from './Link';
import css from './ResourcePoolCard.module.scss';
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
  { key: 'location', tag: 'Location' },
  { key: 'instanceType', tag: 'Instance Type' },
  { key: 'preemptible', tag: 'Spot/Preemptible' },
  { key: 'minAgents', tag: 'Min Agents' },
  { key: 'maxAgents', tag: 'Max Agents' },
  { key: 'slotsPerAgent', tag: 'Slots Per Agent' },
  { key: 'auxContainerCapacityPerAgent', tag: 'Max Aux Containers Per Agent' },
  { key: 'schedulerType', tag: 'Scheduler Type' },
];

type SafeRawJson = Record<string, unknown>;

const ResourcePoolCard: React.FC<Props> = ({
  computeContainerStates,
  resourcePool: pool,
  totalComputeSlots,
  resourceType,
}: Props) => {
  const [ detailVisible, setDetailVisible ] = useState(false);
  const statusClasses = [ css.agentsStatus ];
  const descriptionClasses = [ css.description ];

  if (pool.numAgents !== 0) statusClasses.push(css.active);
  if (!pool.description) descriptionClasses.push(css.empty);

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
      const value = processedPool[attribute.key as keyof ResourcePool];
      acc[attribute.tag] = value;
      return acc;
    }, {} as SafeRawJson);
  }, [ processedPool ]);

  const tags: string[] = [ V1ResourcePoolTypeToLabel[pool.type] ];
  if (pool.defaultComputePool) tags.push('default compute pool');
  if (pool.defaultAuxPool) tags.push('default aux pool');

  const toggleModal = useCallback(() => setDetailVisible((cur: boolean) => !cur), []);

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.icon}>{poolLogo(pool.type)}</div>
        <div className={css.info}>
          <div className={css.name}>{pool.name}</div>
          <div className={css.tags}>
            {tags.map(tag => (
              <Badge key={tag} type={BadgeType.Header}>{tag.toUpperCase()}</Badge>
            ))}
          </div>
        </div>
      </div>
      <div className={statusClasses.join(' ')}>
        <p>{`${pool.numAgents || 'No'} Connected Agent${pool.numAgents > 1 ? 's' : ''}`}</p>
      </div>
      <div className={css.body}>
        <section className={descriptionClasses.join(' ')}>
          <p className={css.fade}>{pool.description || 'No description.'}</p>
        </section>
        <hr />
        <section>
          {totalComputeSlots > 0 && (
            <SlotAllocationBar
              resourceStates={computeContainerStates}
              title={deviceTypes.has(resourceType) ? resourceType : undefined}
              totalSlots={totalComputeSlots}
            />
          )}
          {pool.auxContainerCapacityPerAgent > 0 && (
            <div className={css.cpuContainers}>
              <span>Aux containers running:</span>
              <span>{pool.auxContainersRunning}</span>
            </div>
          )}
        </section>
        <hr />
        <section className={css.details}>
          <Json json={shortDetails} />
          <div>
            <Link onClick={toggleModal}>View more info</Link>
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

export default ResourcePoolCard;
