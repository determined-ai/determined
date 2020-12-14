import React from 'react';

import awsLogo from 'assets/aws-logo.svg';
import gcpLogo from 'assets/gcp-logo.svg';
import staticLogo from 'assets/on-prem-logo.svg';
import Badge, { BadgeType } from 'components/Badge';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { getResourcePools } from 'services/api';
import { getStateColorCssVar } from 'themes';
import { Agent } from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import Json from './Json';
import Link from './Link';
import css from './ResourcePoolCard.module.scss';

interface Props {
  agents: Agent[];
}

const resoucePools = getResourcePools();

const rpAttrs = [
  'location' ,
  'instanceType',
  'spotOrPreemptible',
  'minInstances',
  'maxInstances',
  'gpusPerAgent',
  'cpuContainerCapacityPerAgent',
  'schedulerType',
];

type SafeRawJson = Record<string, unknown>;

const agentStatusText = (numAgents: number, maxInstances: number): string => {
  let prefix = '';
  if (numAgents === 0) {
    prefix = 'No';
  } else if (maxInstances === 0) {
    prefix = numAgents + '';

  } else {
    prefix = `${numAgents}/${maxInstances}`;
  }
  return prefix + ' Agents Active';
};

const ResourcePoolCard: React.FC<Props> = ({ agents }: Props) => {
  const rp = resoucePools[Math.floor(
    Math.random() * resoucePools.length,
  )];

  const shortDetails = rpAttrs.reduce((acc, cur) => {
    acc[cur] = (rp as SafeRawJson) [cur];
    return acc;
  }, {} as SafeRawJson );

  const {
    name,
    description,
    type,
    gpusPerAgent,
    numAgents,
  } = rp;

  let iconSrc = staticLogo;
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

    default:
      console.error('unexpected resource pool type');
      break;
  }

  const slotStates = getSlotContainerStates(agents, name);

  const tags: string[] = [ type ];
  if (rp.defaultGpuPool) tags.push('default gpu pool');
  if (rp.defaultCpuPool) tags.push('default cpu pool');

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.icon}>
          <img src={iconSrc} />
        </div>
        <div className={css.info}>
          <div className={css.name}>{name}</div>
          <div className={css.tags}>
            {tags.map(tag => (
              <Badge bgColor="#132231" key={tag} type={BadgeType.Custom}>{tag.toUpperCase()}</Badge>
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
          {agentStatusText(numAgents, rp.maxInstances)}
        </p>
      </div>
      <div className={css.body}>
        <section className={css.description}>
          <p>
            {description}
          </p>
        </section>
        <hr />
        <section>
          <SlotAllocationBar resourceStates={slotStates} totalSlots={numAgents * gpusPerAgent} />
          <div className={css.spaceBetweenHorizontal}>
            <span>CPU containers running:</span>
            <span>{rp.cpuContainersRunning}</span>
          </div>
        </section>
        <hr />
        <section className={css.details}>
          <Json json={shortDetails} />
          <div>
            <Link>View more info</Link>
          </div>
        </section>
        <div />
      </div>
    </div>
  );
};

export default ResourcePoolCard;
