import { Card } from 'antd';
import React from 'react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { getResourcePools } from 'services/api';
import { Agent } from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import Link from './Link';
import css from './ResourcePoolCard.module.scss';

const resoucePools = getResourcePools();

interface Props {
  agents: Agent[];
}

const ResourcePoolCard: React.FC<Props> = ({ agents }: Props) => {
  const classes = [ css.base ];

  const {
    name,
    description,
    type,
    gpusPerAgent,
    defaultGpuPool,
    numAgents,
    ...rp
  } = resoucePools[Math.floor(
    Math.random() * resoucePools.length,
  )];

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
              <span>Default pool: {defaultGpuPool || false}</span>
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
        <div>
          <SlotAllocationBar resourceStates={slotStates} totalSlots={numAgents * gpusPerAgent} />
        </div>
        <div>
          <ul>
            <li>{rp.location}</li>
            <li>{rp.instanceType}</li>
            <li>{rp.spotOrPreemptible}</li>
            <li>{rp.minInstances}</li>
            <li>{rp.maxInstances}</li>
            <li>deet 2</li>
            <li>deet 3</li>
          </ul>
        </div>
        <div>
          <Link>View more info</Link>
        </div>
      </div>
    </Card>
  );
};

export default ResourcePoolCard;
