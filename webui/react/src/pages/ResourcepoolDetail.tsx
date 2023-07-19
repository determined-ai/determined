import type { TabsProps } from 'antd';
import { Divider } from 'antd';
import React, { Fragment, Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Json from 'components/Json';
import Empty from 'components/kit/Empty';
import Pivot from 'components/kit/Pivot';
import Page from 'components/Page';
import ResourcePoolBindings from 'components/ResourcePoolBindings';
import { RenderAllocationBarResourcePool } from 'components/ResourcePoolCard';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import { V1SchedulerTypeToLabel } from 'constants/states';
import useFeature from 'hooks/useFeature';
import { paths } from 'routes/utils';
import { getJobQStats } from 'services/api';
import { V1GetJobQueueStatsResponse, V1RPQueueStat, V1SchedulerType } from 'services/api-ts-sdk';
import clusterStore from 'stores/cluster';
import { maxPoolSlotCapacity } from 'stores/cluster';
import determinedStore from 'stores/determinedInfo';
import { ShirtSize } from 'themes';
import { ValueOf } from 'types';
import { JobState, ResourceState } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { clone } from 'utils/data';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { camelCaseToSentence, floatToPercent } from 'utils/string';

import ClustersQueuedChart from './Clusters/ClustersQueuedChart';
import JobQueue from './JobQueue/JobQueue';
import css from './ResourcepoolDetail.module.scss';

type Params = {
  poolname?: string;
  tab?: TabType;
};

const TabType = {
  Active: 'active',
  Bindings: 'Bindings',
  Configuration: 'configuration',
  Queued: 'queued',
  Stats: 'stats',
} as const;

type TabType = ValueOf<typeof TabType>;

export const DEFAULT_POOL_TAB_KEY = TabType.Active;

const ResourcepoolDetailInner: React.FC = () => {
  const { poolname, tab } = useParams<Params>();
  const rpBindingFlagOn = useFeature().isOn('rp_binding');
  const { rbacEnabled } = useObservable(determinedStore.info);
  const agents = Loadable.getOrElse([], useObservable(clusterStore.agents));
  const resourcePools = useObservable(clusterStore.resourcePools);

  const pool = useMemo(() => {
    if (Loadable.isLoading(resourcePools)) return;

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
    const details = clone(pool.details);
    for (const key in details) {
      if (details[key] === null) {
        delete details[key];
      }
    }

    const mainSection = clone(pool);
    delete mainSection.details;
    delete mainSection.stats;
    return (
      <>
        <Json alternateBackground json={mainSection} translateLabel={camelCaseToSentence} />
        {Object.keys(details).map((key) => (
          <Fragment key={key}>
            <Divider />
            <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
            <Json alternateBackground json={details[key]} translateLabel={camelCaseToSentence} />
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
        children: <ClustersQueuedChart poolStats={poolStats} />,
        key: TabType.Stats,
        label: 'Stats',
      },
      {
        children: renderPoolConfig(),
        key: TabType.Configuration,
        label: 'Configuration',
      },
    ];

    if (rpBindingFlagOn && rbacEnabled) {
      tabItems.push({
        children: <ResourcePoolBindings poolName={pool.name} />,
        key: TabType.Bindings,
        label: 'Bindings',
      });
    }

    return tabItems;
  }, [pool, poolStats, rbacEnabled, renderPoolConfig, rpBindingFlagOn]);

  if (!pool || Loadable.isLoading(resourcePools)) return <Spinner center spinning />;

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
        <Section>
          {pool.schedulerType === V1SchedulerType.ROUNDROBIN ? (
            <Section>
              <Empty description="Resource Pool is unavailable for Round Robin schedulers." />
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
  <Suspense fallback={<Spinner />}>
    <ResourcepoolDetailInner />
  </Suspense>
);

export default ResourcepoolDetail;
