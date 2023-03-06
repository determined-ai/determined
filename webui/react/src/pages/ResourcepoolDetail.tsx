import type { TabsProps } from 'antd';
import { Divider } from 'antd';
import React, { Fragment, Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Json from 'components/Json';
import Pivot from 'components/kit/Pivot';
import Page from 'components/Page';
import { PoolLogo, RenderAllocationBarResourcePool } from 'components/ResourcePoolCard';
import Section from 'components/Section';
import { V1SchedulerTypeToLabel } from 'constants/states';
import { paths } from 'routes/utils';
import { getJobQStats } from 'services/api';
import { V1GetJobQueueStatsResponse, V1RPQueueStat, V1SchedulerType } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { ValueOf } from 'shared/types';
import { clone } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { camelCaseToSentence, floatToPercent } from 'shared/utils/string';
import { useClusterStore } from 'stores/cluster';
import { ShirtSize } from 'themes';
import { JobState, ResourceState } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import ClustersQueuedChart from './Clusters/ClustersQueuedChart';
import { maxPoolSlotCapacity } from './Clusters/utils';
import JobQueue from './JobQueue/JobQueue';
import css from './ResourcepoolDetail.module.scss';

type Params = {
  poolname?: string;
  tab?: TabType;
};

const TabType = {
  Active: 'active',
  Configuration: 'configuration',
  Queued: 'queued',
  Stats: 'stats',
} as const;

type TabType = ValueOf<typeof TabType>;

export const DEFAULT_POOL_TAB_KEY = TabType.Active;

const ResourcepoolDetailInner: React.FC = () => {
  const { poolname, tab } = useParams<Params>();
  const resourcePools = Loadable.getOrElse([], useObservable(useClusterStore().resourcePools)); // TODO show spinner when this is loading
  const agents = Loadable.getOrElse([], useObservable(useClusterStore().agents));

  const pool = useMemo(() => {
    return resourcePools.find((pool) => pool.name === poolname);
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
      <Page bodyNoPadding>
        <Json alternateBackground json={mainSection} translateLabel={camelCaseToSentence} />
        {Object.keys(details).map((key) => (
          <Fragment key={key}>
            <Divider />
            <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
            <Json alternateBackground json={details[key]} translateLabel={camelCaseToSentence} />
          </Fragment>
        ))}
      </Page>
    );
  }, [pool]);

  const tabItems: TabsProps['items'] = useMemo(() => {
    if (!pool) {
      return [];
    }

    return [
      {
        children: <JobQueue bodyNoPadding jobState={JobState.SCHEDULED} selectedRp={pool} />,
        key: TabType.Active,
        label: `${poolStats?.stats.scheduledCount ?? ''} Active`,
      },
      {
        children: <JobQueue bodyNoPadding jobState={JobState.QUEUED} selectedRp={pool} />,
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
  }, [pool, poolStats, renderPoolConfig]);

  if (!pool) return <div />;

  return (
    <Page className={css.poolDetailPage}>
      <Section>
        <div className={css.nav} onClick={() => navigate(paths.cluster(), { replace: true })}>
          <Icon name="arrow-left" size="tiny" />
          <div className={css.icon}>
            <PoolLogo type={pool.type} />
          </div>
          <div>
            {`${pool.name} (${V1SchedulerTypeToLabel[pool.schedulerType]}) ${
              usage ? `- ${floatToPercent(usage)}` : ''
            } `}
          </div>
        </div>
      </Section>
      <Section>
        <RenderAllocationBarResourcePool
          poolStats={poolStats}
          resourcePool={pool}
          size={ShirtSize.Large}
        />
      </Section>
      <Section>
        {pool.schedulerType === V1SchedulerType.ROUNDROBIN ? (
          <Page className={css.poolDetailPage}>
            <Section>
              <Message
                title="Resource Pool is unavailable for Round Robin schedulers."
                type={MessageType.Empty}
              />
            </Section>
          </Page>
        ) : (
          <Pivot
            activeKey={tabKey}
            destroyInactiveTabPane={true}
            items={tabItems}
            onChange={handleTabChange}
          />
        )}
      </Section>
    </Page>
  );
};

const ResourcepoolDetail: React.FC = () => (
  <Suspense fallback={<Spinner />}>
    <ResourcepoolDetailInner />
  </Suspense>
);

export default ResourcepoolDetail;
