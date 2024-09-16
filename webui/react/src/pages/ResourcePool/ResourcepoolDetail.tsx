import Divider from 'hew/Divider';
import { MenuItem } from 'hew/Dropdown';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Pivot, { PivotProps } from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import { ShirtSize } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';
import { isEmpty } from 'lodash';
import React, { Fragment, Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import JsonGlossary from 'components/JsonGlossary';
import ManageNodesModalComponent from 'components/ManageNodesModal';
import Page from 'components/Page';
import ResourcePoolBindings from 'components/ResourcePoolBindings';
import { RenderAllocationBarResourcePool } from 'components/ResourcePoolCard';
import Section from 'components/Section';
import { V1SchedulerTypeToLabel } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import ClusterQueuedChart from 'pages/Cluster/ClusterQueuedChart';
import JobQueue from 'pages/JobQueue/JobQueue';
import Topology from 'pages/ResourcePool/ClusterTopology';
import { paths } from 'routes/utils';
import { getAgents, getJobQStats } from 'services/api';
import { V1ResourcePoolDetail, V1RPQueueStat, V1SchedulerType } from 'services/api-ts-sdk';
import clusterStore, { maxPoolSlotCapacity } from 'stores/cluster';
import { Agent, JobState, JsonObject, ResourceState, ValueOf } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { camelCaseToSentence, floatToPercent } from 'utils/string';

import css from './ResourcepoolDetail.module.scss';

type Params = {
  poolname?: string;
  tab?: TabType;
};

const TabType = {
  Active: 'active',
  Bindings: 'bindings',
  Configuration: 'configuration',
  Queued: 'queued',
  Stats: 'stats',
} as const;

type TabType = ValueOf<typeof TabType>;

export const DEFAULT_POOL_TAB_KEY = TabType.Active;

// PERF: hide topology if nodes or slots are huge to avoid rendering issue
const MAX_USABLE_NODES = 1_000;
const MAX_USABLE_SLOTS = 10_000;

const MenuKey = {
  ManageNodes: 'manage-nodes',
} as const;

const ResourcepoolDetailInner: React.FC = () => {
  const { poolname, tab } = useParams<Params>();
  const rpBindingFlagOn = useFeature().isOn('rp_binding');
  const { canManageResourcePoolBindings } = usePermissions();
  const agents = useObservable(clusterStore.agents);
  const resourcePools = useObservable(clusterStore.resourcePools);
  const navigate = useNavigate();
  const [canceler] = useState(new AbortController());

  const [tabKey, setTabKey] = useState<TabType>(tab ?? DEFAULT_POOL_TAB_KEY);
  const [poolsStats, setPoolsStats] = useState<V1RPQueueStat[]>();
  const [agentsWithSlots, setAgentsWithSlots] = useState<Agent[]>([]);

  const pool = useMemo(() => {
    if (!Loadable.isLoaded(resourcePools)) return;

    return resourcePools.data.find((pool) => pool.name === poolname);
  }, [poolname, resourcePools]);

  const totalSlots = useMemo(() => {
    const total = agents
      .getOrElse([])
      .reduce(
        (totalVal, { slotStats }) =>
          totalVal +
          Object.values(slotStats.typeStats ?? {}).reduce(
            (localTotal, { total }) => localTotal + total,
            0,
          ),
        0,
      );
    return total;
  }, [agents]);

  const isTopologyAvailable =
    agents.isLoaded && agents.data.length <= MAX_USABLE_NODES && totalSlots <= MAX_USABLE_SLOTS;

  const fetchAgentsWithSlots = useCallback(async () => {
    if (!isTopologyAvailable) {
      setAgentsWithSlots([]);
      return;
    }
    try {
      const response = await getAgents({ excludeSlots: false });
      setAgentsWithSlots(response);
    } catch (e) {
      handleError(e, { publicSubject: 'Could not get agents with slots' });
      setAgentsWithSlots([]);
    }
  }, [isTopologyAvailable]);

  const usage = useMemo(() => {
    if (!pool) return 0;
    const totalSlots = pool.slotsAvailable;
    const resourceStates = getSlotContainerStates(agentsWithSlots, pool.slotType, pool.name);
    const runningState = resourceStates.filter((s) => s === ResourceState.Running).length;
    const slotsPotential = maxPoolSlotCapacity(pool);
    const slotsAvaiablePer =
      slotsPotential && slotsPotential > totalSlots ? totalSlots / slotsPotential : 1;
    return totalSlots < 1 ? 0 : (runningState / totalSlots) * slotsAvaiablePer;
  }, [pool, agentsWithSlots]);

  const topologyAgentPool = useMemo(
    () =>
      poolname
        ? agentsWithSlots.filter(({ resourcePools }) => resourcePools.includes(poolname))
        : [],
    [poolname, agentsWithSlots],
  );

  const fetchStats = useCallback(async () => {
    try {
      const stats = await getJobQStats({}, { signal: canceler.signal });
      const pool = stats.results;
      setPoolsStats(pool);
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to fetch job queue stats.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [canceler.signal]);

  const fetchAll = useCallback(async () => {
    await Promise.all([fetchStats(), fetchAgentsWithSlots()]);
  }, [fetchAgentsWithSlots, fetchStats]);

  const { stopPolling } = usePolling(fetchAll, { rerunOnNewFn: true });

  const rpStats = useMemo<V1RPQueueStat[]>(() => {
    if (!Loadable.isLoaded(resourcePools)) return [];

    return resourcePools.data.map((rp) => {
      const matchStats = poolsStats?.find((p) => p.resourcePool === rp.name);
      return {
        resourcePool: rp.name,
        stats: matchStats
          ? matchStats.stats
          : { preemptibleCount: 0, queuedCount: 0, scheduledCount: 0 },
      } as V1RPQueueStat;
    });
  }, [resourcePools, poolsStats]);

  useEffect(() => {
    if (tab || !pool) return;
    const basePath = paths.resourcePool(pool.name);
    navigate(`${basePath}/${DEFAULT_POOL_TAB_KEY}`, { replace: true });
  }, [navigate, pool, tab]);

  useEffect(() => {
    setTabKey(tab ?? DEFAULT_POOL_TAB_KEY);
  }, [tab]);

  useEffect(() => {
    return () => {
      stopPolling();
    };
  }, [stopPolling]);

  const handleTabChange = useCallback(
    (key: string) => {
      if (!pool) return;
      setTabKey(key as TabType);
      const basePath = paths.resourcePool(pool.name);
      navigate(`${basePath}/${key}`);
    },
    [navigate, pool],
  );

  const renderPoolConfig = useCallback(() => {
    if (!pool) return;
    const { details, stats, resourceManagerMetadata, ...mainSection } = structuredClone(pool);
    for (const key in details) {
      if (details[key as keyof V1ResourcePoolDetail] === null) {
        delete details[key as keyof V1ResourcePoolDetail];
      }
    }

    return (
      <>
        <JsonGlossary alignValues="right" json={mainSection} translateLabel={camelCaseToSentence} />
        {!isEmpty(resourceManagerMetadata) && (
          <>
            <Divider />
            <div className={css.subTitle}>Resource Manager Metadata</div>
            <JsonGlossary json={resourceManagerMetadata} translateLabel={camelCaseToSentence} />
          </>
        )}
        {Object.keys(details).map((key) => (
          <Fragment key={key}>
            <Divider />
            <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
            <JsonGlossary
              json={details[key as keyof V1ResourcePoolDetail] as unknown as JsonObject}
              translateLabel={camelCaseToSentence}
            />
          </Fragment>
        ))}
      </>
    );
  }, [pool]);

  const poolStats = useMemo(
    () => poolsStats?.find((p) => p.resourcePool === poolname),
    [poolsStats, poolname],
  );

  const tabItems: PivotProps['items'] = useMemo(() => {
    if (!pool) {
      return [];
    }

    const tabItems: PivotProps['items'] = [
      {
        children: <JobQueue jobState={JobState.SCHEDULED} rpStats={rpStats} selectedRp={pool} />,
        key: TabType.Active,
        label: `${poolStats?.stats.scheduledCount ?? ''} Active`,
      },
      {
        children: <JobQueue jobState={JobState.QUEUED} rpStats={rpStats} selectedRp={pool} />,
        key: TabType.Queued,
        label: `${poolStats?.stats.queuedCount ?? ''} Queued`,
      },
      {
        children: <ClusterQueuedChart poolStats={poolStats} />,
        key: TabType.Stats,
        label: 'Stats',
      },
      {
        children: renderPoolConfig(),
        key: TabType.Configuration,
        label: 'Configuration',
      },
    ];

    if (rpBindingFlagOn && canManageResourcePoolBindings) {
      tabItems.push({
        children: <ResourcePoolBindings pool={pool} />,
        key: TabType.Bindings,
        label: 'Bindings',
      });
    }

    return tabItems;
  }, [canManageResourcePoolBindings, pool, poolStats, renderPoolConfig, rpStats, rpBindingFlagOn]);

  const ManageNodesModal = useModal(ManageNodesModalComponent);

  const menu: MenuItem[] | undefined = useMemo(
    () =>
      canManageResourcePoolBindings
        ? [
            {
              disabled: false,
              key: MenuKey.ManageNodes,
              label: 'Manage Nodes',
            },
          ]
        : undefined,
    [canManageResourcePoolBindings],
  );

  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case MenuKey.ManageNodes:
          ManageNodesModal.open();
          break;
      }
    },
    [ManageNodesModal],
  );

  if (!pool || Loadable.isNotLoaded(resourcePools)) {
    return <Spinner center spinning />;
  } else if (Loadable.isFailed(resourcePools)) {
    return null; // TODO inform user here if resource pools fail to load
  }

  return (
    <Page
      breadcrumb={[
        { breadcrumbName: 'Cluster', path: paths.clusters() },
        {
          breadcrumbName: `${pool.name} (${V1SchedulerTypeToLabel[pool.schedulerType]}) ${
            usage ? `- ${floatToPercent(usage)}` : ''
          }`,
          path: '',
        },
      ]}
      menuItems={menu}
      title={
        tabKey === TabType.Active || tabKey === TabType.Queued
          ? 'Job Queue by Resource Pool'
          : undefined
      }
      onClickMenu={handleDropdown}>
      <div className={css.poolDetailPage}>
        <Section>
          <RenderAllocationBarResourcePool
            poolStats={poolStats}
            resourcePool={pool}
            size={ShirtSize.Large}
          />
        </Section>
        {isTopologyAvailable && (
          <>
            {topologyAgentPool.length !== 0 && poolname && <Topology nodes={topologyAgentPool} />}
          </>
        )}
        <Section>
          {pool.schedulerType === V1SchedulerType.ROUNDROBIN ? (
            <Section>
              <Message description="Resource Pool is unavailable for Round Robin schedulers." />
            </Section>
          ) : (
            <Pivot
              activeKey={tabKey}
              destroyInactiveTabPane={true}
              items={tabItems}
              onChange={handleTabChange}
            />
          )}
        </Section>
        <ManageNodesModal.Component nodes={topologyAgentPool} />
      </div>
    </Page>
  );
};

const ResourcepoolDetail: React.FC = () => (
  <Suspense fallback={<Spinner spinning />}>
    <ResourcepoolDetailInner />
  </Suspense>
);

export default ResourcepoolDetail;
