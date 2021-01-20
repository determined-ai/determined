import React, { useCallback, useState } from 'react';

import awsLogo from 'assets/aws-logo.svg';
import gcpLogo from 'assets/gcp-logo.svg';
import staticLogo from 'assets/on-prem-logo.svg';
import Badge, { BadgeType } from 'components/Badge';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { ResourcePool, ResourceState } from 'types';

import Json from './Json';
import Link from './Link';
import css from './ResourcePoolCard.module.scss';
import ResourcePoolDetails from './ResourcePoolDetails';

interface Props {
  containerStates: ResourceState[]; // GPU
  resourcePool: ResourcePool;
}

export const rpLogo = (type: string): React.ReactNode => {
  let iconSrc = '';
  switch (type) {
    case 'aws':
      iconSrc = awsLogo;
      break;
    case 'gcp':
      iconSrc = gcpLogo;
      break;
    case 'static':
      iconSrc = staticLogo;
      break;
  }

  return <img src={iconSrc} />;
};

const rpAttrs = [
  [ 'location', 'Location' ] ,
  [ 'instanceType', 'Instance Type' ],
  [ 'preemptible', 'Spot/Preemptible' ],
  [ 'minAgents', 'Min Agents' ],
  [ 'maxAgents', 'Max Agents' ],
  [ 'gpusPerAgent', 'GPUs per Agent' ],
  [ 'cpuContainerCapacityPerAgent', 'CPU Containers per Agent' ],
  [ 'schedulerType', 'Scheduler Type' ],
];

type SafeRawJson = Record<string, unknown>;

const agentStatusText = (numAgents: number, maxAgents: number): string => {
  let prefix = '';
  if (numAgents === 0) {
    prefix = 'No';
  } else if (maxAgents === 0) {
    prefix = numAgents + '';

  } else {
    prefix = `${numAgents}/${maxAgents}`;
  }
  return prefix + ' Agents Active';
};

const ResourcePoolCard: React.FC<Props> = ({ containerStates, resourcePool: rp }: Props) => {
  const [ detailVisible, setDetailVisible ] = useState(false);

  const shortDetails = rpAttrs.reduce((acc, cur) => {
    acc[cur[1]] = (rp as unknown as SafeRawJson)[cur[0]];
    return acc;
  }, {} as SafeRawJson );

  const {
    name,
    description,
    type,
    gpusPerAgent,
    numAgents,
  } = rp;

  const tags: string[] = [ type ];
  if (rp.defaultGpuPool) tags.push('default gpu pool');
  if (rp.defaultCpuPool) tags.push('default cpu pool');

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
            {/* QUESTION do we want default gpu or cpu pool */}
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
          {agentStatusText(numAgents, rp.maxAgents)}
        </p>
      </div>
      <div className={css.body}>
        <section className={css.description}>
          <p className={css.fade}>
            {description}
          </p>
        </section>
        <hr />
        <section>
          <SlotAllocationBar
            resourceStates={containerStates}
            totalSlots={numAgents * gpusPerAgent} />
          <div className={css.cpuContainers}>
            <span>CPU containers running:</span>
            <span>{rp.cpuContainersRunning}</span>
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
