import React, { Suspense, useMemo } from 'react';

import Card from 'components/kit/Card';
import Icon from 'components/kit/Icon';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { V1ResourcePoolTypeToLabel, V1SchedulerTypeToLabel } from 'constants/states';
import { paths } from 'routes/utils';
import { V1ResourcePoolType, V1RPQueueStat, V1SchedulerType } from 'services/api-ts-sdk';
import awsLogoOnDark from 'shared/assets/images/aws-logo-on-dark.svg';
import awsLogo from 'shared/assets/images/aws-logo.svg';
import gcpLogo from 'shared/assets/images/gcp-logo.svg';
import k8sLogo from 'shared/assets/images/k8s-logo.svg';
import staticLogo from 'shared/assets/images/on-prem-logo.svg';
import Spinner from 'shared/components/Spinner';
import useUI from 'shared/contexts/stores/UI';
import { DarkLight } from 'shared/themes';
import { clone } from 'shared/utils/data';
import { maxPoolSlotCapacity } from 'stores/cluster';
import clusterStore from 'stores/cluster';
import { ShirtSize } from 'themes';
import { isDeviceType, ResourcePool } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import Json from './Json';
import css from './ResourcePoolCard.module.scss';

interface Props {
  poolStats?: V1RPQueueStat | undefined;
  resourcePool: ResourcePool;
  size?: ShirtSize;
}

const poolAttributes = [
  {
    key: 'accelerator',
    label: 'Accelerator',
    render: (x: ResourcePool) => (x.accelerator ? x.accelerator : '--'),
  },
  {
    key: 'type',
    label: 'Type',
    render: (x: ResourcePool) => V1ResourcePoolTypeToLabel[x.type],
  },
  { key: 'instanceType', label: 'Instance Type' },
  {
    key: 'numAgents',
    label: 'Connected Agents',
    render: (x: ResourcePool) => {
      if (x.maxAgents > 0) {
        return `${x.numAgents}/${x.maxAgents}`;
      }
      return x.numAgents;
    },
  },
  { key: 'slotsPerAgent', label: 'Slots Per Agent' },
  { key: 'auxContainerCapacityPerAgent', label: 'Aux Containers Per Agent' },
  { key: 'schedulerType', label: 'Scheduler Type' },
];

type SafeRawJson = Record<string, unknown>;

/** Resource pool logo based on resource pool type */
export const PoolLogo: React.FC<{ type: V1ResourcePoolType }> = ({ type }) => {
  const { ui } = useUI();

  let iconSrc = '';
  switch (type) {
    case V1ResourcePoolType.AWS:
      iconSrc = ui.darkLight === DarkLight.Light ? awsLogo : awsLogoOnDark;
      break;
    case V1ResourcePoolType.GCP:
      iconSrc = gcpLogo;
      break;
    case V1ResourcePoolType.K8S:
      iconSrc = k8sLogo;
      break;
    case V1ResourcePoolType.UNSPECIFIED:
    case V1ResourcePoolType.STATIC:
      iconSrc = staticLogo;
      break;
  }

  return <img className={css['rp-type-logo']} src={iconSrc} />;
};

const ResourcePoolCard: React.FC<Props> = ({ resourcePool: pool }: Props) => {
  const descriptionClasses = [css.description];

  if (!pool.description) descriptionClasses.push(css.empty);

  const isAux = useMemo(() => {
    return pool.auxContainerCapacityPerAgent > 0;
  }, [pool]);

  const processedPool = useMemo(() => {
    const newPool = clone(pool);
    Object.keys(newPool).forEach((key) => {
      const value = pool[key as keyof ResourcePool];
      if (key === 'slotsPerAgent' && value === -1) newPool[key] = 'Unknown';
      if (key === 'schedulerType') newPool[key] = V1SchedulerTypeToLabel[value as V1SchedulerType];
    });
    return newPool;
  }, [pool]);

  const shortDetails = useMemo(() => {
    return poolAttributes.reduce((acc, attribute) => {
      const value = attribute.render
        ? attribute.render(processedPool)
        : processedPool[attribute.key as keyof ResourcePool];
      acc[attribute.label] = value;
      if (!isAux && attribute.key === 'auxContainerCapacityPerAgent') delete acc[attribute.label];
      if (pool.type === V1ResourcePoolType.K8S && attribute.key !== 'type') {
        delete acc[attribute.label];
      }
      return acc;
    }, {} as SafeRawJson);
  }, [processedPool, isAux, pool]);

  return (
    <Card href={paths.resourcePool(pool.name)} size="medium">
      <div className={css.base}>
        <div className={css.header}>
          <div className={css.info}>
            <div className={css.name}>{pool.name}</div>
          </div>
          <div className={css.default}>
            {(pool.defaultAuxPool || pool.defaultComputePool) && <span>Default</span>}
            {pool.description && <Icon name="info" title={pool.description} />}
          </div>
        </div>
        <Suspense fallback={<Spinner center />}>
          <div className={css.body}>
            <RenderAllocationBarResourcePool resourcePool={pool} size={ShirtSize.Medium} />
            <section className={css.details}>
              <Json hideDivider json={shortDetails} />
            </section>
            <div />
          </div>
        </Suspense>
      </div>
    </Card>
  );
};

export const RenderAllocationBarResourcePool: React.FC<Props> = ({
  poolStats,
  resourcePool: pool,
  size = ShirtSize.Large,
}: Props) => {
  const agents = Loadable.waitFor(useObservable(clusterStore.agents));
  const isAux = useMemo(() => {
    return pool.auxContainerCapacityPerAgent > 0;
  }, [pool]);
  return (
    <section>
      <SlotAllocationBar
        footer={{
          queued: poolStats?.stats.queuedCount ?? pool?.stats?.queuedCount,
          scheduled: poolStats?.stats.scheduledCount ?? pool?.stats?.scheduledCount,
        }}
        hideHeader
        poolName={pool.name}
        poolType={pool.type}
        resourceStates={getSlotContainerStates(agents || [], pool.slotType, pool.name)}
        size={size}
        slotsPotential={maxPoolSlotCapacity(pool)}
        title={isDeviceType(pool.slotType) ? pool.slotType : undefined}
        totalSlots={pool.slotsAvailable}
      />
      {isAux && (
        <SlotAllocationBar
          footer={{
            auxContainerCapacity: pool.auxContainerCapacity,
            auxContainersRunning: pool.auxContainersRunning,
          }}
          hideHeader
          isAux={true}
          poolType={pool.type}
          resourceStates={getSlotContainerStates(agents || [], pool.slotType, pool.name)}
          size={size}
          title={isDeviceType(pool.slotType) ? pool.slotType : undefined}
          totalSlots={maxPoolSlotCapacity(pool)}
        />
      )}
    </section>
  );
};

export default ResourcePoolCard;
