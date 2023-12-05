import { Divider, type TabsProps } from 'antd';
import Message from 'hew/Message';
import Pivot from 'hew/Pivot';
import Spinner from 'hew/Spinner';
import { ShirtSize } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';
import React, { Fragment, Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import JsonGlossary from 'components/JsonGlossary';
import Page from 'components/Page';
import ResourcePoolBindings from 'components/ResourcePoolBindings';
import { RenderAllocationBarResourcePool } from 'components/ResourcePoolCard';
import Section from 'components/Section';
import { V1SchedulerTypeToLabel } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import ClusterQueuedChart from 'pages/Cluster/ClusterQueuedChart';
import JobQueue from 'pages/JobQueue/JobQueue';
import Topology from 'pages/ResourcePool/Topology';
import { paths } from 'routes/utils';
import { getJobQStats } from 'services/api';
import {
  V1GetJobQueueStatsResponse,
  V1ResourcePoolDetail,
  V1RPQueueStat,
  V1SchedulerType,
} from 'services/api-ts-sdk';
import clusterStore, { maxPoolSlotCapacity } from 'stores/cluster';
import { JobState, JsonObject, ResourceState, ValueOf } from 'types';
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

const ResourcepoolDetailInner: React.FC = () => {
  const { poolname, tab } = useParams<Params>();
  const rpBindingFlagOn = useFeature().isOn('rp_binding');
  const { canManageResourcePoolBindings } = usePermissions();
  const agents = Loadable.getOrElse([], useObservable(clusterStore.agents));
  const resourcePools = useObservable(clusterStore.resourcePools);
  const pool = useMemo(() => {
    if (!Loadable.isLoaded(resourcePools)) return;

    return resourcePools.data.find((pool) => pool.name === poolname);
  }, [poolname, resourcePools]);

  const usage = useMemo(() => {
    if (!pool) return 0;
    const totalSlots = pool.slotsAvailable;
    const resourceStates = getSlotContainerStates(agents || [], pool.slotType, pool.name);
    const runningState = resourceStates.filter((s) => s === ResourceState.Running).length;
    const slotsPotential = maxPoolSlotCapacity(pool);
    const slotsAvaiablePer =
      slotsPotential && slotsPotential > totalSlots ? totalSlots / slotsPotential : 1;
    return totalSlots < 1 ? 0 : (runningState / totalSlots) * slotsAvaiablePer;
  }, [pool, agents]);

  const navigate = useNavigate();
  const [canceler] = useState(new AbortController());

  const [tabKey, setTabKey] = useState<TabType>(tab ?? DEFAULT_POOL_TAB_KEY);
  const [poolStats, setPoolStats] = useState<V1RPQueueStat>();

  const topologyAgentPool = useMemo(
    () => (poolname ? agents.filter(({ resourcePools }) => resourcePools.includes(poolname)) : []),
    [poolname, agents],
  );

  const fetchStats = useCallback(async () => {
    try {
      const promises = [getJobQStats({}, { signal: canceler.signal })] as [
        Promise<V1GetJobQueueStatsResponse>,
      ];
      const [stats] = await Promise.all(promises);
      const pool = stats.results.find((p) => p.resourcePool === poolname);
      setPoolStats(pool);
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to fetch job queue stats.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [canceler.signal, poolname]);

  useEffect(() => {
    fetchStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (tab || !pool) return;
    const basePath = paths.resourcePool(pool.name);
    navigate(`${basePath}/${DEFAULT_POOL_TAB_KEY}`, { replace: true });
  }, [navigate, pool, tab]);

  useEffect(() => {
    setTabKey(tab ?? DEFAULT_POOL_TAB_KEY);
  }, [tab]);

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
    const { details, stats, ...mainSection } = structuredClone(pool);
    for (const key in details) {
      if (details[key as keyof V1ResourcePoolDetail] === null) {
        delete details[key as keyof V1ResourcePoolDetail];
      }
    }

    return (
      <>
        <JsonGlossary alignValues="right" json={mainSection} translateLabel={camelCaseToSentence} />
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

  const tabItems: TabsProps['items'] = useMemo(() => {
    if (!pool) {
      return [];
    }

    const tabItems: TabsProps['items'] = [
      {
        children: <JobQueue jobState={JobState.SCHEDULED} selectedRp={pool} />,
        key: TabType.Active,
        label: `${poolStats?.stats.scheduledCount ?? ''} Active`,
      },
      {
        children: <JobQueue jobState={JobState.QUEUED} selectedRp={pool} />,
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
  }, [canManageResourcePoolBindings, pool, poolStats, renderPoolConfig, rpBindingFlagOn]);

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
      title={
        tabKey === TabType.Active || tabKey === TabType.Queued
          ? 'Job Queue by Resource Pool'
          : undefined
      }>
      <div className={css.poolDetailPage}>
        <Section>
          <RenderAllocationBarResourcePool
            poolStats={poolStats}
            resourcePool={pool}
            size={ShirtSize.Large}
          />
        </Section>
        {!!topologyAgentPool.length && poolname && <Topology nodes={topologyAgentPool} />}
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
