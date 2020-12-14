import { Card } from 'antd';
import React from 'react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { getResourcePools } from 'services/api';
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

const ResourcePoolCard: React.FC<Props> = ({ agents }: Props) => {
  const classes = [ css.base ];

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

  const slotStates = getSlotContainerStates(agents, name);

  const tags: string[] = [ type ];
  if (rp.defaultGpuPool) tags.push('default gpu pool');
  if (rp.defaultCpuPool) tags.push('default cpu pool');

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.icon}><Avatar name={type} /></div>
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
      <div className={css.agentsStatus}>
        <p>
          {numAgents}/{rp.maxInstances} Agents Active
        </p>
      </div>
      <div className={css.body}>
        <section>{description}</section>
        <hr />
        <section>
          <SlotAllocationBar resourceStates={slotStates} totalSlots={numAgents * gpusPerAgent} />
          <div> CPU containers running: {rp.cpuContainersRunning} </div>
        </section>
        <hr />
        <section>
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
