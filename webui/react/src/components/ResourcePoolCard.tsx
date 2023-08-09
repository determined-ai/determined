import React, { Suspense, useCallback, useEffect, useMemo } from 'react';

import awsLogoOnDark from 'assets/images/aws-logo-on-dark.svg?url';
import awsLogo from 'assets/images/aws-logo.svg?url';
import gcpLogo from 'assets/images/gcp-logo.svg?url';
import k8sLogo from 'assets/images/k8s-logo.svg?url';
import staticLogo from 'assets/images/on-prem-logo.svg?url';
import Card from 'components/kit/Card';
import { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Spinner from 'components/kit/Spinner';
import SlotAllocationBar from 'components/SlotAllocationBar';
import { V1ResourcePoolTypeToLabel, V1SchedulerTypeToLabel } from 'constants/states';
import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { V1ResourcePoolType, V1RPQueueStat, V1SchedulerType } from 'services/api-ts-sdk';
import { maxPoolSlotCapacity } from 'stores/cluster';
import clusterStore from 'stores/cluster';
import useUI from 'stores/contexts/UI';
import workspaceStore from 'stores/workspaces';
import { ShirtSize } from 'themes';
import { isDeviceType, JsonObject, ResourcePool } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { clone } from 'utils/data';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { DarkLight } from 'utils/themes';

import Json from './Json';
import ResourcePoolBindingModalComponent from './ResourcePoolBindingModal';
import css from './ResourcePoolCard.module.scss';

interface Props {
  actionMenu?: MenuItem[];
  poolStats?: V1RPQueueStat | undefined;
  resourcePool: ResourcePool;
  size?: ShirtSize;
  descriptiveLabel?: string;
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

const ResourcePoolCard: React.FC<Props> = ({
  resourcePool: pool,
  actionMenu,
  descriptiveLabel,
}: Props) => {
  const rpBindingFlagOn = useFeature().isOn('rp_binding');
  const ResourcePoolBindingModal = useModal(ResourcePoolBindingModalComponent);

  const descriptionClasses = [css.description];

  const resourcePoolBindingMap = useObservable(clusterStore.resourcePoolBindings);
  const resourcePoolBindings: number[] = resourcePoolBindingMap.get(pool.name, []);
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  useEffect(() => {
    return clusterStore.fetchResourcePoolBindings(pool.name);
  }, [pool.name]);

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
    }, {} as JsonObject);
  }, [processedPool, isAux, pool]);

  const onDropdown = useCallback(
    (key: string) => {
      if (key === 'bindings') {
        ResourcePoolBindingModal.open();
      }
    },
    [ResourcePoolBindingModal],
  );

  const onSaveBindings = useCallback(
    (bindings: string[]) => {
      const workspaceIds = workspaces.filter((w) => bindings.includes(w.name)).map((w) => w.id);
      clusterStore.overwriteResourcePoolBindings(pool.name, workspaceIds);
    },
    [workspaces, pool],
  );

  return (
    <Card
      actionMenu={actionMenu}
      href={paths.resourcePool(pool.name)}
      size="medium"
      onDropdown={onDropdown}>
      <div className={css.base}>
        <div className={css.header}>
          <div className={css.info}>
            <div className={css.name}>{pool.name}</div>
          </div>
          <div className={css.default}>
            <span>{descriptiveLabel}</span>
            {pool.description && <Icon name="info" showTooltip title={pool.description} />}
          </div>
        </div>
        <Suspense fallback={<Spinner center spinning />}>
          <div className={css.body}>
            <RenderAllocationBarResourcePool resourcePool={pool} size={ShirtSize.Medium} />
            {rpBindingFlagOn && resourcePoolBindings.length > 0 && (
              <section className={css.resoucePoolBoundContainer}>
                <div>Bound to:</div>
                <div className={css.resoucePoolBoundCount}>
                  <Icon name="lock" title="Bound Workspaces" />
                  {resourcePoolBindings.length} workspace
                </div>
              </section>
            )}
            <section className={css.details}>
              <Json hideDivider json={shortDetails} />
            </section>
            <div />
          </div>
        </Suspense>
      </div>
      <ResourcePoolBindingModal.Component
        bindings={workspaces.filter((w) => resourcePoolBindings.includes(w.id)).map((w) => w.name)}
        pool={pool.name}
        workspaces={workspaces.map((w) => w.name)}
        onSave={onSaveBindings}
      />
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
