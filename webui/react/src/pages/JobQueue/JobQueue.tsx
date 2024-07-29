import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import { DetError } from 'hew/utils/error';
import { Loadable } from 'hew/utils/loadable';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import ActionDropdown, { Triggers } from 'components/ActionDropdown';
import Section from 'components/Section';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  checkmarkRenderer,
  createOmitableRenderer,
  defaultRowClassName,
  getFullPaginationConfig,
  userRenderer,
} from 'components/Table/Table';
import { V1SchedulerTypeToLabel } from 'constants/states';
import useFeature from 'hooks/useFeature';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { columns as columnsFunc, SCHEDULING_VAL_KEY } from 'pages/JobQueue/JobQueue.table';
import { paths } from 'routes/utils';
import { cancelExperiment, getJobQ, killExperiment, killTask } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import userStore from 'stores/users';
import { DetailedUser, FullJob, Job, JobAction, JobState, JobType, ResourcePool } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { canManageJob, jobTypeToCommandType, orderedSchedulers } from 'utils/job';
import { useObservable } from 'utils/observable';
import { routeToReactUrl } from 'utils/routes';
import { numericSorter } from 'utils/sort';
import { capitalize } from 'utils/string';

import css from './JobQueue.module.scss';
import settingsConfig, { Settings } from './JobQueue.settings';
import ManageJobModalComponent from './ManageJob';

interface Props {
  jobState: JobState;
  rpStats: Api.V1RPQueueStat[];
  selectedRp: ResourcePool;
}

const JobQueue: React.FC<Props> = ({ rpStats, selectedRp, jobState }) => {
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const [managingJob, setManagingJob] = useState<Job>();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [topJob, setTopJob] = useState<Job>();
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const [pageState, setPageState] = useState<{ isLoading: boolean }>({ isLoading: true });
  const manageJobModal = useModal(ManageJobModalComponent);
  const pageRef = useRef<HTMLElement>(null);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const defaultColumns = useMemo(() => columnsFunc(f_flat_runs), [f_flat_runs]);

  useEffect(() => {
    if (managingJob) {
      manageJobModal.open();
    }
  }, [managingJob, manageJobModal]);
  const { settings, updateSettings } = useSettings<Settings>(
    useMemo(() => settingsConfig(jobState), [jobState]),
  );
  const settingsColumns = useMemo(() => [...settings.columns], [settings.columns]);

  const isJobOrderAvailable = orderedSchedulers.has(selectedRp.schedulerType);

  const fetchJobsTable = useCallback(async () => {
    if (!settings) return;

    try {
      const orderBy = settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC';
      const jobs = await getJobQ(
        {
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy,
          resourcePool: selectedRp.name,
          states: jobState ? [jobState] : undefined,
        },
        { signal: canceler.signal },
      );

      const firstJobResp = await getJobQ(
        {
          limit: 1,
          offset: 0,
          resourcePool: selectedRp.name,
        },
        { signal: canceler.signal },
      );
      const firstJob = firstJobResp.jobs[0];

      // Process jobs response.
      if (firstJob && !_.isEqual(firstJob, topJob)) setTopJob(firstJob);
      setJobs(jobState ? jobs.jobs.filter((j) => j.summary.state === jobState) : jobs.jobs);
      if (jobs.pagination.total !== undefined) setTotal(jobs.pagination.total);
    } catch (e) {
      if ((e as DetError)?.publicMessage === 'offset out of bounds' && settings.tableOffset !== 0) {
        updateSettings({ tableOffset: 0 });
        return;
      }
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to fetch job queue and stats.',
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      setPageState((cur) => ({ ...cur, isLoading: false }));
    }
  }, [canceler.signal, selectedRp.name, settings, jobState, topJob, updateSettings]);

  usePolling(fetchJobsTable, { rerunOnNewFn: true });

  const dropDownOnTrigger = useCallback(
    (job: Job) => {
      if (!('entityId' in job)) return {};
      const triggers: Triggers<JobAction> = {};
      const commandType = jobTypeToCommandType(job.type);

      if (commandType) {
        triggers[JobAction.Kill] = () => {
          killTask({ id: job.entityId, type: commandType });
        };
        triggers[JobAction.ViewLog] = () => {
          routeToReactUrl(paths.taskLogs({ id: job.entityId, name: job.name, type: commandType }));
        };
      }

      // if job is an experiment type add action to kill it
      if (job.type === JobType.EXPERIMENT) {
        triggers[JobAction.Cancel] = async () => {
          await cancelExperiment({ experimentId: parseInt(job.entityId, 10) });
        };
        triggers[JobAction.Kill] = async () => {
          await killExperiment({ experimentId: parseInt(job.entityId, 10) });
        };
      }

      if (canManageJob(job, selectedRp)) {
        triggers[JobAction.ManageJob] = () => setManagingJob(job);
      }

      Object.keys(triggers).forEach((key) => {
        const action = key as JobAction;
        const fn = triggers[action];
        if (!fn) return;
        triggers[action] = async () => {
          await fn();
          await fetchJobsTable();
        };
      });
      return triggers;
    },
    [selectedRp, fetchJobsTable],
  );

  const onModalClose = useCallback(() => {
    setManagingJob(undefined);
    fetchJobsTable();
  }, [fetchJobsTable]);

  useEffect(() => {
    if (!managingJob) return;
    const job = jobs.find((j) => j.jobId === managingJob.jobId);
    if (!job) {
      setManagingJob(undefined);
    } else if (!_.isEqual(job, managingJob)) {
      setManagingJob(job);
    }
  }, [jobs, managingJob]);

  useEffect(() => {
    const col = defaultColumns.find(({ key }) => key === SCHEDULING_VAL_KEY);
    if (col) {
      const replaceIndex = settingsColumns.findIndex((column) =>
        ['priority', 'weight', 'resourcePool'].includes(column),
      );
      const newColumns = [...settingsColumns];
      if (replaceIndex !== -1) newColumns[replaceIndex] = col.dataIndex;
      if (!_.isEqual(newColumns, settings.columns)) updateSettings({ columns: newColumns });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings.columns, settingsColumns]);

  const columns = useMemo(() => {
    return defaultColumns
      .map<ColumnDef<Job>>((col) => {
        switch (col.key) {
          case 'actions':
            return {
              ...col,
              render: createOmitableRenderer<Job, FullJob>(
                'entityId',
                (_, record) => (
                  <div>
                    <ActionDropdown<JobAction>
                      actionOrder={[
                        JobAction.ManageJob,
                        JobAction.ViewLog,
                        JobAction.Cancel,
                        JobAction.Kill,
                      ]}
                      confirmations={{
                        [JobAction.Cancel]: { cancelText: 'Abort', onError: handleError },
                        [JobAction.Kill]: { danger: true, onError: handleError },
                      }}
                      id={record.name}
                      kind="job"
                      onError={handleError}
                      onTrigger={dropDownOnTrigger(record)}
                    />
                  </div>
                ),
                null,
              ),
            };
          case SCHEDULING_VAL_KEY: {
            if (!settingsColumns) return col;

            switch (selectedRp.schedulerType) {
              case Api.V1SchedulerType.SLURM:
                return {
                  ...col,
                  dataIndex: 'resourcePool',
                  title: 'Partition',
                };
              case Api.V1SchedulerType.PBS:
                return {
                  ...col,
                  dataIndex: 'resourcePool',
                  title: 'Queue',
                };
              case Api.V1SchedulerType.PRIORITY:
              case Api.V1SchedulerType.KUBERNETES:
                return {
                  ...col,
                  dataIndex: 'priority',
                  title: 'Priority',
                };
              case Api.V1SchedulerType.FAIRSHARE:
                return {
                  ...col,
                  align: 'right',
                  dataIndex: 'weight',
                  title: 'Weight',
                };
              case Api.V1SchedulerType.UNSPECIFIED:
              case Api.V1SchedulerType.ROUNDROBIN:
                return col;
            }
          }
          case 'jobsAhead':
            if (!isJobOrderAvailable) {
              return {
                ...col,
                render: (_, record) => (
                  <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
                    {checkmarkRenderer(record.isPreemptible)}
                  </div>
                ),
                title: 'Preemptible',
              };
            } else {
              return {
                ...col,
                render: (_: unknown, record) => (
                  <div className={css.centerVertically}>
                    {record.summary.jobsAhead}
                    {!record.isPreemptible && <Icon name="lock" title="Not Preemptible" />}
                  </div>
                ),
                sorter: (a, b) => numericSorter(a.summary.jobsAhead, b.summary.jobsAhead),
                title: '#',
              };
            }
          case 'user':
            return {
              ...col,
              render: createOmitableRenderer<Job, FullJob>('entityId', (_, r) => {
                let user = users.find((u) => u.id === r.userId);
                if (!user) {
                  // This is an external user. Create a new DetailedUser instance.
                  const externalUser: DetailedUser = {
                    // external users do not have a user id. Indicate that with a value of -1.
                    id: -1,
                    isActive: true,
                    isAdmin: false,
                    username: r.username,
                  };
                  user = externalUser;
                }
                return userRenderer(user);
              }),
            };
          default:
            return col;
        }
      })
      .map((column) => {
        column.sortOrder = null;
        if (column.key === settings.sortKey) {
          column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
        }
        return column;
      });

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    isJobOrderAvailable,
    dropDownOnTrigger,
    settingsColumns,
    settings.sortKey,
    settings.sortDesc,
    selectedRp.schedulerType,
  ]);

  // table title using selectedRp and schedulerType from list of resource pools
  const tableTitle = useMemo(() => {
    if (!selectedRp) return '';
    const schedulerType = V1SchedulerTypeToLabel[selectedRp.schedulerType];
    return (
      <div>
        {`${capitalize(selectedRp.name)} (${schedulerType.toLowerCase()}) `}
        <Icon name="info" title={`Job Queue for resource pool "${selectedRp.name}"`} />
      </div>
    );
  }, [selectedRp]);

  return (
    <div className={css.base} id="jobs">
      <Section hideTitle={!!selectedRp} title={tableTitle}>
        {settings ? (
          <InteractiveTable<Job, Settings>
            columns={columns}
            containerRef={pageRef}
            dataSource={jobs}
            loading={pageState.isLoading}
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              total,
            )}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="jobId"
            scroll={{ x: 1000 }}
            settings={settings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings}
          />
        ) : (
          <SkeletonTable columns={columns.length} />
        )}
      </Section>
      {!!managingJob && (
        <manageJobModal.Component
          initialPool={selectedRp.name}
          job={managingJob}
          rpStats={rpStats}
          schedulerType={selectedRp.schedulerType}
          onFinish={onModalClose}
        />
      )}
    </div>
  );
};

export default JobQueue;
