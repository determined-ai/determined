import { Divider, Tabs } from 'antd';
import React, { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Icon from 'components/Icon';
import Json from 'components/Json';
import Page from 'components/Page';
import { poolLogo } from 'components/ResourcePoolCard';
import { RenderAllocationBarResourcePool } from 'components/ResourcePoolCardLight';
import Section from 'components/Section';
import { V1SchedulerTypeToLabel } from 'constants/states';
import { useStore } from 'contexts/Store';
import usePolling from 'hooks/usePolling';
import { maxPoolSlotCapacity } from 'pages/Cluster/ClusterOverview';
import { paths } from 'routes/utils';
import { getJobQStats } from 'services/api';
import { V1GetJobQueueStatsResponse, V1RPQueueStat } from 'services/api-ts-sdk';
import { clone } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { camelCaseToSentence } from 'shared/utils/string';
import { floatToPercent } from 'shared/utils/string';
import { ShirtSize } from 'themes';
import { ResourceState } from 'types';
import { JobState } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import handleError from 'utils/error';

import ClustersQueuedChart from './Clusters/ClustersQueuedChart';
import JobQueue from './JobQueue/JobQueue';
import css from './ResourcepoolDetail.module.scss';

interface Params {
  poolname?: string;
  tab?: TabType;
}
const { TabPane } = Tabs;

enum TabType {
  Active = 'active',
  Queued = 'queued',
  Stats = 'stats',
  Configuration = 'configuration'
}
const DEFAULT_TAB_KEY = TabType.Active;

const ResourcepoolDetail: React.FC = () => {

  const { poolname } = useParams<Params>();
  const { agents, resourcePools } = useStore();

  const pool = useMemo(() => {
    return resourcePools.find(pool => pool.name === poolname);
  }, [ poolname, resourcePools ]);

  const usage = useMemo(() => {
    if(!pool) return 0;
    const totalSlots = pool.slotsAvailable;
    const resourceStates = getSlotContainerStates(agents || [], pool.slotType, pool.name);
    const runningState = resourceStates.filter(s => s === ResourceState.Running).length;
    const slotsPotential = maxPoolSlotCapacity(pool);
    const slotsAvaiablePer = slotsPotential && slotsPotential > totalSlots
      ? (totalSlots / slotsPotential) : 1;
    return totalSlots < 1 ? 0 : (runningState / totalSlots) * slotsAvaiablePer;
  }, [ pool, agents ]);

  const { tab } = useParams<Params>();

  const history = useHistory();
  const [ canceler ] = useState(new AbortController());

  const [ tabKey, setTabKey ] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const [ poolStats, setPoolStats ] = useState<V1RPQueueStat>();

  const fetchStats = useCallback(async () => {
    try {
      const promises = [
        getJobQStats({}, { signal: canceler.signal }),
      ] as [ Promise<V1GetJobQueueStatsResponse> ];
      const [ stats ] = await Promise.all(promises);
      const pool = stats.results.find(p => p.resourcePool === poolname);
      setPoolStats(pool);

    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to fetch job queue stats.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ canceler.signal, poolname ]);

  usePolling(fetchStats);

  useEffect(() => {
    fetchStats();
    return () => canceler.abort();
  }, [ canceler, fetchStats ]);

  const handleTabChange = useCallback(key => {
    if(!pool) return;
    setTabKey(key);
    const basePath = paths.resourcePool(pool.name);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ history, pool ]);

  const renderPoolConfig = useCallback(() => {
    if(!pool) return;
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
      <Page>
        <Json alternateBackground json={mainSection} translateLabel={camelCaseToSentence} />
        {Object.keys(details).map(key => (
          <Fragment key={key}>
            <Divider />
            <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
            <Json alternateBackground json={details[key]} translateLabel={camelCaseToSentence} />
          </Fragment>
        ))}
      </Page>
    );
  }, [ pool ]);

  if(!pool) return <div />;
  return (
    <Page className={css.poolDetailPage}>
      <Section>
        <div className={css.nav} onClick={() => history.replace(paths.cluster())}>
          <Icon name="arrow-left" size="tiny" />
          <div className={css.icon}>{poolLogo(pool.type)}</div>
          <div>{`${pool.name} (${
            V1SchedulerTypeToLabel[pool.schedulerType]
          }) ${usage ? `- ${floatToPercent(usage)}` : '' } `}
          </div>
        </div>
      </Section>
      <Section>
        <RenderAllocationBarResourcePool
          poolStats={poolStats}
          resourcePool={pool}
          size={ShirtSize.large}
        />
      </Section>
      <Section>
        <Tabs
          className="no-padding"
          defaultActiveKey={tabKey}
          destroyInactiveTabPane={true}
          onChange={handleTabChange}>
          <TabPane key="active" tab={`${poolStats?.stats.scheduledCount} Active`}>
            <JobQueue bodyNoPadding jobState={JobState.SCHEDULED} selected={pool} />
          </TabPane>
          <TabPane key="queued" tab={`${poolStats?.stats.queuedCount} Queued`}>
            <JobQueue bodyNoPadding jobState={JobState.QUEUED} selected={pool} />
          </TabPane>
          <TabPane key="stats" tab="Stats">
            <ClustersQueuedChart poolStats={poolStats} />
          </TabPane>
          <TabPane key="configuration" tab="Configuration">
            {renderPoolConfig()}
          </TabPane>
        </Tabs>
      </Section>

    </Page>
  );

};

export default ResourcepoolDetail;
