import { Popover } from 'antd';
import React, { useMemo } from 'react';

import Badge from 'components/Badge';
import Bar from 'components/Bar';
import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Link from 'components/Link';
import { resourceStateToLabel } from 'constants/states';
import { paths } from 'routes/utils';
import { V1ResourcePoolType } from 'services/api-ts-sdk';
import { floatToPercent } from 'shared/utils/string';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { ResourceState, SlotState } from 'types';

import { BadgeType } from './Badge';
import css from './SlotAllocation.module.scss';

export interface Props {
  barOnly?: boolean;
  className?: string;
  footer?: AllocationBarFooterProps;
  hideHeader?: boolean;
  isAux?: boolean;
  poolType?: V1ResourcePoolType;
  resourceStates: ResourceState[];
  showLegends?: boolean;
  size?: ShirtSize;
  slotsPotential?:number;
  title?: string;
  totalSlots: number;
}

export interface AllocationBarFooterProps {
  auxContainerCapacity?:number
  auxContainersRunning?: number;
  queued?: number;
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
  footer,
  isAux,
  title,
  poolType,
  slotsPotential,
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
    if (isAux && footer) {
      const freePerc = footer.auxContainerCapacity && footer.auxContainersRunning &&
      footer.auxContainerCapacity - footer.auxContainersRunning > 0 ?
        (footer.auxContainerCapacity - footer.auxContainersRunning) / footer.auxContainerCapacity
        : 1;
      const parts = {
        free: {
          color: getStateColorCssVar(SlotState.Free),
          percent: freePerc,
        },
        running: {
          color: getStateColorCssVar(SlotState.Running),
          percent: 1 - freePerc,
        },
      };
      return [ parts.running, parts.free ];
    }
    const slotsAvaiablePer = slotsPotential && slotsPotential > totalSlots
      ? (totalSlots / slotsPotential) : 1;
    const parts = {
      free: {
        color: getStateColorCssVar(SlotState.Free),
        percent: totalSlots < 1 ? 0 : (freeSlots / totalSlots) * slotsAvaiablePer,
      },
      pending: {
        color: getStateColorCssVar(SlotState.Pending),
        percent: totalSlots < 1 ? 0 : (pendingSlots / totalSlots) * slotsAvaiablePer,
      },
      potential: {
        bordered: true,
        color: getStateColorCssVar(SlotState.Potential),
        percent: 1 - slotsAvaiablePer,
      },
      running: {
        color: getStateColorCssVar(SlotState.Running),
        percent: totalSlots < 1 ? 0 : (stateTallies.RUNNING / totalSlots) * slotsAvaiablePer,
      },
    };

    return [ parts.running, parts.pending, parts.free, parts.potential ];
  }, [ totalSlots, stateTallies, pendingSlots, freeSlots, slotsPotential, footer, isAux ]);

  const stateDetails = useMemo(() => {
    const states = [
      ResourceState.Assigned,
      ResourceState.Pulling,
      ResourceState.Starting,
      ResourceState.Running,
    ];
    return (
      <ul className={css.detailedLegends}>
        {states.map((state) => (
          <Legend count={stateTallies[state]} key={state} totalSlots={totalSlots}>
            <Badge
              state={state === ResourceState.Running ? SlotState.Running : SlotState.Pending}
              type={BadgeType.State}>
              {resourceStateToLabel[state]}
            </Badge>
          </Legend>
        ))}
      </ul>
    );
  }, [ stateTallies, totalSlots ]);

  const classes = [ css.base ];
  if (className) classes.push(className);

  return (
    <div className={classes.join(' ')}>
      {!hideHeader && (
        <div className={css.header}>
          <header>{title || 'Compute'} Slots Allocated</header>
          {totalSlots === 0 ? <span>0/0</span> : (
            <span>
              {resourceStates.length}/{totalSlots}
              {totalSlots > 0 ? ` (${floatToPercent(resourceStates.length / totalSlots, 0)})` : ''}
            </span>
          )}
        </div>
      )}
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
      {footer && (
        <div className={css.footer}>
          {poolType === V1ResourcePoolType.K8S ? (
            <header>{`${isAux ?
              `${footer.auxContainersRunning} Aux Containers Running` :
              `${stateTallies.RUNNING} Compute Slots Allocated`}`}
            </header>
          )
            : (
              <header>{`${isAux ?
                `${footer.
                  auxContainersRunning}/${footer.auxContainerCapacity} Aux Containers Running` :
                `${stateTallies.RUNNING}/${totalSlots} Compute Slots Allocated`}`}
              </header>
            )}
          {footer.queued ? (
            <Link path={paths.jobs()}>
              <span className={css.queued}>{`${footer.queued > 100 ?
                '100+' :
                footer.queued} ${footer.queued === 1 ? 'Job' : 'Jobs'} Queued`}
              </span>
            </Link>
          ) :
            !isAux && <span>{`${totalSlots - resourceStates.length} Slots Free`}</span>}
        </div>
      )}
      {showLegends && (
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
      )}
    </div>
  );
};

export default SlotAllocationBar;
