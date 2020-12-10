import { Card } from 'antd';
import React from 'react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import { getResourcePools } from 'services/api';

import css from './ResourcePoolCard.module.scss';

const resoucePools = getResourcePools();

const ResourcePoolCard: React.FC = () => {
  const classes = [ css.base ];

  const {
    name,
    maxInstances,
    description, type,
    defaultGpuPool, numAgents,
  } = resoucePools[Math.floor(
    Math.random() * resoucePools.length,
  )];

  const iconName = 'experiment';

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
        {numAgents}/{maxInstances} Agents Active
      </div>
      <div className={css.lower}>
        <div>{description}</div>
        <div>slots chart</div>
        <div>deets</div>
        <div>view more info</div>
      </div>
    </Card>
  );
};

export default ResourcePoolCard;
