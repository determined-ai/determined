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
    defaultGpuPool,
    numAgents,
  } = rp;

  const slotStates = getSlotContainerStates(agents, name);

  return (
    <Card
      className={classes.join(' ')}
      title={
        <div className={css.upper}>
          <div className={css.icon}><Avatar name={type} /></div>
          <div className={css.info}>
            <div className={css.name}>{name}</div>
            <div className={css.tags}>
              <span>{type}</span>
              {/* QUESTION is this default gpu or cpu pool */}
              <span>Default GPU pool: {(!!defaultGpuPool).toString()}</span>
              {/* TODO custom badge */}
            </div>
          </div>
        </div>
      }>
      <div className={css.agentsStatus}>
        {numAgents}/{rp.maxInstances} Agents Active
      </div>
      <div className={css.lower}>
        <div>{description}</div>
        <hr />
        <div>
          <SlotAllocationBar resourceStates={slotStates} totalSlots={numAgents * gpusPerAgent} />
          <div> CPU containers running: {rp.cpuContainersRunning} </div>
        </div>
        <hr />
        <div>
          <Json json={shortDetails} />
        </div>
        <div>
          <Link>View more info</Link>
        </div>
      </div>
    </Card>
  );
};

export default ResourcePoolCard;
