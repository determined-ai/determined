import React, { useCallback, useState } from 'react';

import awsLogo from 'assets/images/aws-logo.svg';
import gcpLogo from 'assets/images/gcp-logo.svg';
import k8sLogo from 'assets/images/k8s-logo.svg';
import staticLogo from 'assets/images/on-prem-logo.svg';
import Badge, { BadgeType } from 'components/Badge';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { V1ResourcePoolTypeToLabel, V1SchedulerTypeToLabel } from 'constants/states';
import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import { deviceTypes, ResourcePool, ResourceState, ResourceType } from 'types';

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

export const rpLogo = (type: V1ResourcePoolType): React.ReactNode => {
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

const rpAttrs = [
  [ 'location', 'Location' ],
  [ 'instanceType', 'Instance Type' ],
  [ 'preemptible', 'Spot/Preemptible' ],
  [ 'minAgents', 'Min Agents' ],
  [ 'maxAgents', 'Max Agents' ],
  [ 'slotsPerAgent', 'Slots Per Agent' ],
  [ 'auxContainerCapacityPerAgent', 'Max Aux Containers Per Agent' ],
  [ 'schedulerType', 'Scheduler Type' ],
];

type SafeRawJson = Record<string, unknown>;

const agentStatusText = (numAgents: number): string => {
  let prefix = '';
  if (numAgents === 0) {
    prefix = 'No';
  } else {
    prefix = numAgents + '';
  }
  return prefix + ' Connected Agent' + (numAgents > 1 ? 's' : '');
};

const ResourcePoolCard: React.FC<Props> = (
  { computeContainerStates, resourcePool: rp, totalComputeSlots, resourceType }: Props,
) => {
  const [ detailVisible, setDetailVisible ] = useState(false);

  const shortDetails = rpAttrs.reduce((acc, cur) => {
    acc[cur[1]] = (rp as unknown as SafeRawJson)[cur[0]];
    return acc;
  }, {} as SafeRawJson);
  shortDetails['Scheduler Type'] =
    V1SchedulerTypeToLabel[shortDetails['Scheduler Type'] as V1SchedulerType];

  const {
    name,
    description,
    type,
    numAgents,
  } = rp;

  const descriptionClasses = [ css.description ];
  if (!description) descriptionClasses.push(css.empty);

  const tags: string[] = [ V1ResourcePoolTypeToLabel[type] ];
  if (rp.defaultComputePool) tags.push('default compute pool');
  if (rp.defaultAuxPool) tags.push('default aux pool');

  const toggleModal = useCallback(
    () => {
      setDetailVisible((cur: boolean) => !cur);
    },
    [],
  );

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.icon}>{rpLogo(rp.type)}</div>
        <div className={css.info}>
          <div className={css.name}>{name}</div>
          <div className={css.tags}>
            {tags.map(tag => (
              <Badge key={tag} type={BadgeType.Header}>{tag.toUpperCase()}</Badge>
            ))}
          </div>
        </div>
      </div>
      <div
        className={css.agentsStatus}
        style={{
          backgroundColor: numAgents > 0 ?
            'var(--theme-colors-states-active)' : 'var(--theme-colors-states-inactive)',
        }}>
        <p>
          {agentStatusText(numAgents)}
        </p>
      </div>
      <div className={css.body}>
        <section className={descriptionClasses.join(' ')}>
          <p className={css.fade}>
            {description || 'No description provided.'}
          </p>
        </section>
        <hr />
        <section>
          <SlotAllocationBar
            resourceStates={computeContainerStates}
            title={deviceTypes.has(resourceType) ? resourceType : undefined }
            totalSlots={totalComputeSlots} />
          <div className={css.cpuContainers}>
            <span>Aux containers running:</span>
            <span>{rp.auxContainersRunning}</span>
          </div>
        </section>
        <hr />
        <section className={css.details}>
          <Json json={shortDetails} />
          <div>
            <Link onClick={toggleModal}>View more info</Link>
            <ResourcePoolDetails finally={toggleModal} resourcePool={rp} visible={detailVisible} />
          </div>
        </section>
        <div />
      </div>
    </div>
  );
};

export default ResourcePoolCard;
