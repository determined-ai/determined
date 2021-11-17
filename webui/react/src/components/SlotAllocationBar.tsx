import { Popover } from 'antd';
import React, { useMemo } from 'react';

import Badge from 'components/Badge';
import Bar from 'components/Bar';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { ResourceState, SlotState } from 'types';
import { ConditionalWrapper } from 'utils/react';
import { floatToPercent } from 'utils/string';
import { resourceStateToLabel } from 'utils/types';

import { BadgeType } from './Badge';
import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
  className?: string;
  hideHeader?: boolean;
  resourceStates: ResourceState[];
  showLegends?: boolean;
  size?: ShirtSize;
  title?: string;
  totalSlots: number;
}

interface LegendProps {
  children: React.ReactNode;
  count: number;
  showPercentage?: boolean;
  totalSlots: number;
}

const Legend: React.FC<LegendProps> = ({
  count, totalSlots,
  showPercentage, children,
}: LegendProps) => {

  let label = `0 (${floatToPercent(0, 0)})`;
  if (totalSlots !== 0) {
    label = count.toString();
    if (showPercentage) label += ` (${floatToPercent(count / totalSlots, 0)})`;
  }
  return (
    <li className={css.legend}>
      <span className={css.count}>
        {label}
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
  hideHeader,
  title,
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

  const barParts = useMemo(() => {
    const parts = {
      free: {
        color: getStateColorCssVar(SlotState.Free),
        percent: totalSlots < 1 ? 0 : freeSlots / totalSlots,
      },
      pending: {
        color: getStateColorCssVar(SlotState.Pending),
        percent: totalSlots < 1 ? 0 : pendingSlots / totalSlots,
      },
      running: {
        color: getStateColorCssVar(SlotState.Running),
        percent: totalSlots < 1 ? 0 : stateTallies.RUNNING / totalSlots,
      },
    };

    return [ parts.running, parts.pending, parts.free ];
  }, [ totalSlots, stateTallies, pendingSlots, freeSlots ]);

  const stateDetails = useMemo(() => {
    const states = [
      ResourceState.Assigned,
      ResourceState.Pulling,
      ResourceState.Starting,
      ResourceState.Running,
    ];
    return (
      <ul className={css.detailedLegends}>
        {states.map((state) =>
          <Legend count={stateTallies[state]} key={state} totalSlots={totalSlots}>
            <Badge
              state={state === ResourceState.Running ? SlotState.Running : SlotState.Pending}
              type={BadgeType.State}>
              {resourceStateToLabel[state]}
            </Badge>
          </Legend>)}
      </ul>
    );
  }, [ stateTallies, totalSlots ]);

  const classes = [ css.base ];
  if (className) classes.push(className);

  return (
    <div className={classes.join(' ')}>
      {!hideHeader &&
        <div className={css.header}>
          <header>{title || 'Compute'} Slots Allocated</header>
          {totalSlots === 0 ? <span>0/0</span> :
            <span>
              {resourceStates.length}/{totalSlots}
              {totalSlots > 0 ? ` (${floatToPercent(resourceStates.length / totalSlots, 0)})` : ''}
            </span>
          }
        </div>
      }
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
                  <Badge state={SlotState.Running} type={BadgeType.State} />
                </Legend>
                <Legend count={pendingSlots} showPercentage totalSlots={totalSlots}>
                  <Badge state={SlotState.Pending} type={BadgeType.State} />
                </Legend>
                <Legend count={freeSlots} showPercentage totalSlots={totalSlots}>
                  <Badge state={SlotState.Free} type={BadgeType.State} />
                </Legend>
              </ol>
            </Popover>
          </div>
      }
    </div>
  );
};

export default SlotAllocationBar;
