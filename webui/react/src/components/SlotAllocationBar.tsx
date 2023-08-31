import React, { useMemo } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Bar from 'components/Bar';
import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Tooltip from 'components/kit/Tooltip';
import css from 'components/SlotAllocation.module.scss';
import { resourceStateToLabel } from 'constants/states';
import { paths } from 'routes/utils';
import { V1ResourcePoolType } from 'services/api-ts-sdk';
import { getStateColorCssVar, ShirtSize } from 'themes';
import { ResourceState, SlotState } from 'types';
import { routeToReactUrl } from 'utils/routes';
import { floatToPercent } from 'utils/string';

export interface Props {
  barOnly?: boolean;
  className?: string;
  footer?: AllocationBarFooterProps;
  hideHeader?: boolean;
  isAux?: boolean;
  poolName?: string;
  poolType?: V1ResourcePoolType;
  resourceStates: ResourceState[];
  showLegends?: boolean;
  size?: ShirtSize;
  slotsPotential?: number;
  title?: string;
  totalSlots: number;
}

export interface AllocationBarFooterProps {
  auxContainerCapacity?: number;
  auxContainersRunning?: number;
  queued?: number;
  scheduled?: number;
}

interface LegendProps {
  children: React.ReactNode;
  count: number;
  showPercentage?: boolean;
  totalSlots: number;
}

const Legend: React.FC<LegendProps> = ({
  count,
  totalSlots,
  showPercentage,
  children,
}: LegendProps) => {
  let label = `0 (${floatToPercent(0, 0)})`;
  if (totalSlots !== 0) {
    label = count.toString();
    if (showPercentage) label += ` (${floatToPercent(count / totalSlots, 0)})`;
  }
  return (
    <li className={css.legend}>
      <span className={css.count}>{label}</span>
      <span>{children}</span>
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
  poolName,
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
      [ResourceState.Warm]: 0,
      [ResourceState.Terminated]: 0,
      [ResourceState.Unspecified]: 0,
      [ResourceState.Potential]: 0,
    };
    resourceStates.forEach((state) => {
      tally[state] += 1;
    });
    tally[ResourceState.Warm] = totalSlots - tally[ResourceState.Running];
    tally[ResourceState.Potential] = slotsPotential ? slotsPotential - totalSlots : 0;
    return tally;
  }, [resourceStates, totalSlots, slotsPotential]);

  const freeSlots = totalSlots - resourceStates.length;
  const pendingSlots = resourceStates.length - stateTallies.RUNNING;

  const barParts = useMemo(() => {
    if (isAux && footer) {
      const freePerc =
        footer.auxContainerCapacity &&
        footer.auxContainersRunning &&
        footer.auxContainerCapacity - footer.auxContainersRunning > 0
          ? (footer.auxContainerCapacity - footer.auxContainersRunning) /
            footer.auxContainerCapacity
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
      return [parts.running, parts.free];
    }
    const slotsAvaiablePer =
      slotsPotential && slotsPotential > totalSlots ? totalSlots / slotsPotential : 1;
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

    return [parts.running, parts.pending, parts.free, parts.potential];
  }, [totalSlots, stateTallies, pendingSlots, freeSlots, slotsPotential, footer, isAux]);

  const totalSlotsNum = useMemo(() => {
    return slotsPotential || totalSlots;
  }, [totalSlots, slotsPotential]);

  const states = useMemo(() => {
    let states = [
      ResourceState.Potential,
      ResourceState.Warm,
      ResourceState.Assigned,
      ResourceState.Pulling,
      ResourceState.Starting,
      ResourceState.Running,
    ];
    if (showLegends) {
      states = states.slice(2);
    } else if (stateTallies[ResourceState.Potential] <= 0) {
      states = states.slice(1);
    }
    return states;
  }, [showLegends, stateTallies]);

  const hasLegend = useMemo(() => {
    return states.map((s) => stateTallies[s]).reduce((res, i) => res + i, 0) > 0;
  }, [stateTallies, states]);

  const classes = [css.base];
  if (className) classes.push(className);

  const renderStateDetails = () => {
    return (
      <ul className={css.detailedLegends}>
        {states.map((state) =>
          stateTallies[state] ? (
            <Legend count={stateTallies[state]} key={state} totalSlots={totalSlots}>
              <Badge state={state} type={BadgeType.State}>
                {resourceStateToLabel[state]}
              </Badge>
            </Legend>
          ) : null,
        )}
      </ul>
    );
  };

  const onClickQueued = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    poolName && routeToReactUrl(`${paths.resourcePool(poolName)}/queued`);
  };

  const onClickScheduled = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    poolName && routeToReactUrl(`${paths.resourcePool(poolName)}`);
  };

  const renderFooterJobs = () => {
    if (footer?.queued || footer?.scheduled) {
      return footer.queued ? (
        <div onClick={onClickQueued}>
          <span className={css.queued}>
            {`${footer.queued > 100 ? '100+' : footer.queued} Queued`}
          </span>
        </div>
      ) : (
        <div onClick={onClickScheduled}>
          <span className={css.queued}>
            {`${footer.scheduled && footer.scheduled > 100 ? '100+' : footer.scheduled} Active`}
          </span>
        </div>
      );
    }
    return !isAux && <span>{`${freeSlots} ${freeSlots > 1 ? 'Slots' : 'Slot'} Free`}</span>;
  };

  const renderLegend = () => (
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
  );

  return (
    <div className={classes.join(' ')}>
      {!hideHeader && (
        <div className={css.header}>
          <header>{title || 'Compute'} Slots Allocated</header>
          {totalSlots === 0 ? (
            <span>0/0</span>
          ) : (
            <span>
              {resourceStates.length}/{totalSlots}
              {totalSlots > 0 ? ` (${floatToPercent(resourceStates.length / totalSlots, 2)})` : ''}
            </span>
          )}
        </div>
      )}
      <ConditionalWrapper
        condition={!showLegends}
        wrapper={(ch) =>
          !isAux && hasLegend ? (
            <Tooltip content={renderStateDetails()} placement="bottom">
              {ch}
            </Tooltip>
          ) : (
            <div>{ch}</div>
          )
        }>
        <div className={css.bar}>
          <Bar {...barProps} inline parts={barParts} />
        </div>
      </ConditionalWrapper>
      {footer && (
        <div className={css.footer}>
          {poolType === V1ResourcePoolType.K8S ? (
            <header>
              {`${
                isAux
                  ? `${footer.auxContainersRunning} Aux Containers Running`
                  : `${resourceStates.length} ${title || 'Compute'} Slots Allocated`
              }`}
            </header>
          ) : (
            <header>
              {`${
                isAux
                  ? `${footer.auxContainersRunning}/${footer.auxContainerCapacity} Aux Containers Running`
                  : `${resourceStates.length}/${totalSlotsNum} ${
                      title || 'Compute'
                    } Slots Allocated`
              }`}
            </header>
          )}
          {renderFooterJobs()}
        </div>
      )}
      {showLegends && (
        <div className={css.overallLegends}>
          {hasLegend ? (
            <Tooltip content={renderStateDetails()} placement="bottom">
              {renderLegend()}
            </Tooltip>
          ) : (
            renderLegend()
          )}
        </div>
      )}
    </div>
  );
};

export default SlotAllocationBar;
