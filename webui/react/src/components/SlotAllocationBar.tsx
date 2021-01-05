import { Popover } from 'antd';
import React, { useMemo } from 'react';

import Badge from 'components/Badge';
import Bar from 'components/Bar';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { ResourceState } from 'types';
import { ConditionalWrapper } from 'utils/react';
import { floatToPercent } from 'utils/string';

import { BadgeType } from './Badge';
import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
  className?: string;
  resourceStates: ResourceState[];
  showLegends?: boolean;
  size?: ShirtSize;
  totalSlots: number;
}

interface LegendProps {
  children: React.ReactNode;
  count: number;
  totalSlots: number;
  showPercentage?: boolean;
}

const Legend: React.FC<LegendProps> = ({
  count, totalSlots,
  showPercentage, children,
}: LegendProps) => {

  return (
    <li className={css.legend}>
      <span className={css.count}>
        {count} {showPercentage && `(${floatToPercent(count/totalSlots, 0)})`}
      </span>
      <span>
        {children}
      </span>
    </li>
  );
};

const SlotAllocationBar: React.FC<Props> = ({
  resourceStates,
  totalSlots,
  showLegends,
  className,
  ...barProps
}: Props) => {

  const stateTallies = useMemo(() => {
    const tally: Record<ResourceState, number> = {
      [ResourceState.Assigned]: 0,
      [ResourceState.Pulling]: 0,
      [ResourceState.Running]: 0,
      [ResourceState.Starting]: 0,
      [ResourceState.Terminated]: 0,
      [ResourceState.Unspecified]: 0,
    };
    resourceStates.forEach(state => {
      tally[state] += 1;
    });
    return tally;
  }, [ resourceStates ]);

  const freeSlots = (totalSlots - resourceStates.length);
  const pendingSlots = (resourceStates.length - stateTallies.RUNNING);

  const { barParts, legendParts } = useMemo(() => {

    const parts = {
      free: {
        color: 'var(--theme-colors-monochrome-15)',
        percent: freeSlots / totalSlots,
      },
      pending: {
        color: '#6666CC',
        percent: pendingSlots / totalSlots,
      },
      running: {
        color: getStateColorCssVar(ResourceState.Running),
        percent: stateTallies.RUNNING / totalSlots,
      },
    };

    return {
      barParts: [ parts.running, parts.pending, parts.free ],
      legendParts: parts,
    };
  }, [ totalSlots, stateTallies, pendingSlots, freeSlots ]);

  const stateDetails = useMemo(() => {
    return (
      <ul className={css.detailedLegends}>
        {Object.entries(stateTallies).map(([ state, count ]) =>
          <Legend count={count} key={state} totalSlots={totalSlots}>
            <Badge state={state as ResourceState} type={BadgeType.State} />
          </Legend>)}
      </ul>
    );
  }, [ stateTallies, totalSlots ]);

  const classes = [ css.base ];
  if (className) classes.push(className);

  return (
    <div className={classes.join(' ')}>
      <div className={css.header}>
        <header>GPU Slots Allocated</header>
        {totalSlots === 0 ? <span>0/0</span> :
          <span>
            {resourceStates.length}/{totalSlots}
            {totalSlots > 0 ? ` (${floatToPercent( resourceStates.length/totalSlots, 0)})` : ''}
          </span>
        }
      </div>
      <ConditionalWrapper
        condition={!showLegends}
        wrapper={(ch) => (
          <Popover content={stateDetails} placement="bottom">
            {ch}
          </Popover>
        )}>
        <div className={css.bar}>
          <Bar {...barProps} parts={barParts} />
        </div>
      </ConditionalWrapper>
      {showLegends &&
          <div className={css.overallLegends}>
            <Popover content={stateDetails} placement="bottom">
              <ol>
                <Legend count={stateTallies.RUNNING} showPercentage totalSlots={totalSlots}>
                  <Badge bgColor={legendParts.running.color} type={BadgeType.Custom}>
                Running
                  </Badge>
                </Legend>
                <Legend count={pendingSlots} showPercentage totalSlots={totalSlots}>
                  <Badge bgColor={legendParts.pending.color} type={BadgeType.Custom}>
                Pending
                  </Badge>
                </Legend>
                <Legend count={freeSlots} showPercentage totalSlots={totalSlots}>
                  <Badge bgColor={legendParts.free.color} fgColor="#234B65" type={BadgeType.Custom}>
                Free
                  </Badge>
                </Legend>
              </ol>
            </Popover>
          </div>
      }
    </div>
  );
};

export default SlotAllocationBar;
