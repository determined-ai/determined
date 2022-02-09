import { Form, FormInstance, Input, List, Modal, Select, Typography } from 'antd';
import React, { ReactNode, useCallback, useMemo, useRef } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import { useStore } from 'contexts/Store';
import { columns } from 'pages/JobQueue/JobQueue.table';
import { updateJobQueue } from 'services/api';
import * as api from 'services/api-ts-sdk';
import { Job, JobType, RPStats } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { moveJobToPositionUpdate, orderedSchedulers } from 'utils/job';
import { floatToPercent, truncate } from 'utils/string';

import css from './ManageJob.module.scss';
const { Option } = Select;

interface Props {
  job: Job;
  jobs: Job[];
  onFinish?: () => void;
  schedulerType: api.V1SchedulerType;
  selectedRPStats: RPStats;
}

/*
FormValues capture different adjustable parameters for a job.
position: The position of the job in the queue. (1-based)
resourcePool: The resource pool to run the job on.
priority: The desired priority of the job.
weight: The desired weight of the job.
*/
interface FormValues {
  position: string;
  priority?: string;
  resourcePool: string;
  weight?: string;
}

const formValuesToUpdate = (
  values: FormValues,
  job: Job,
  jobs: Job[],
): api.V1QueueControl | undefined => {
  const { position, resourcePool } = {
    position: parseInt(values.position, 10),
    resourcePool: values.resourcePool,
  };
  const update: api.V1QueueControl = { jobId: job.jobId };

  if (resourcePool !== job.resourcePool) {
    return { ...update, resourcePool };
  }
  if (position !== job.summary.jobsAhead + 1) {
    return moveJobToPositionUpdate(jobs, job.jobId, position);
  }
  if (values.priority !== undefined) {
    const priority = parseInt(values.priority, 10);
    if (priority !== job.priority) {
      return { ...update, priority };
    }
  }
  if (values.weight !== undefined) {
    const weight = parseFloat(values.weight);
    if (weight !== job.weight) {
      return { ...update, weight };
    }
  }
};

const ManageJob: React.FC<Props> = ({ onFinish, selectedRPStats, job, jobs, schedulerType }) => {
  const formRef = useRef <FormInstance<FormValues>>(null);
  const isOrderedQ = orderedSchedulers.has(schedulerType);
  const { resourcePools } = useStore();

  const details = useMemo(() => {
    interface Item {
      label: ReactNode;
      value: ReactNode;
    }
    const tableKeys = [
      'user',
      'slots',
      'submitted',
      'type',
      'name',
    ];
    const tableDetails: Record<string, Item> = {};

    tableKeys.forEach(td => {
      const col = columns.find(col => col.key === td);
      if (!col || !col.render) return;
      tableDetails[td] = { label: col.title, value: col.render(undefined, job, 0) };
    });

    const items = [
      tableDetails.type,
      tableDetails.name,
      { label: 'UUID', value: job.jobId },
      tableDetails.submitted,
      {
        label: 'State',
        value: <Badge state={job.summary.state} type={BadgeType.State} />,
      },
      {
        label: 'Progress',
        value: job.progress ?
          floatToPercent(job.progress, 1) : undefined,
      },
      tableDetails.slots,
      { label: 'Is Preemtible', value: job.isPreemptible ? 'Yes' : 'No' },
      {
        label: 'Jobs Ahead',
        value: isOrderedQ ? job.summary.jobsAhead : undefined,
      },
      tableDetails.user,
    ];

    return items.filter(item => !!item && item.value !== undefined) as Item[];

  }, [ job, isOrderedQ ]);

  const currentPool = useMemo(() => {
    return resourcePools.find(rp => rp.name === selectedRPStats.resourcePool);
  }, [ resourcePools, selectedRPStats.resourcePool ]);

  const poolDetails = useMemo(() => {
    return (
      <div>
        <p>Current slot allocation:{' '}
          {currentPool?.slotsUsed} / {currentPool?.slotsAvailable} (used / total)
          <br />
          Jobs in queue:
          {selectedRPStats.stats.queuedCount + selectedRPStats.stats.scheduledCount}
          <br />
          Spot instance pool: {!!currentPool?.details.aws?.spotEnabled + ''}
        </p>
      </div>
    );
  }, [ currentPool, selectedRPStats ]);

  const onOk = useCallback(
    async () => {
      try{
        const update = formRef.current &&
          formValuesToUpdate(formRef.current.getFieldsValue(), job, jobs);
        if (update) await updateJobQueue({ updates: [ update ] });
      } catch (e) {
        handleError(e, {
          isUserTriggered: true,
          publicSubject: 'Failed to update the job.',
          silent: false,
          type: ErrorType.Api,
        });
      }
      onFinish?.();
    },
    [ formRef, onFinish, job, jobs ],
  );

  const isSingular = job.summary && job.summary.jobsAhead === 1;

  return (
    <Modal
      mask
      title={'Manage Job ' + truncate(job.jobId, 6, '')}
      visible={true}
      onCancel={onFinish}
      onOk={onOk}>
      {isOrderedQ && (
        <p>There {isSingular ? 'is' : 'are'} {job.summary?.jobsAhead || 'no'} job
          {isSingular ? '' : 's'} ahead of this job.
        </p>
      )}
      <h6>
        Queue Settings
      </h6>
      <Form<FormValues>
        initialValues={{
          position: job.summary.jobsAhead + 1,
          priority: job.priority,
          resourcePool: selectedRPStats.resourcePool,
          weight: job.weight,
        }}
        labelCol={{ span: 6 }}
        name="form basic"
        ref={formRef}>
        <Form.Item
          extra="Priority is a whole number from 1 to 99 with 1 being the highest priority."
          hidden={schedulerType !== api.V1SchedulerType.PRIORITY}
          label="Priority"
          name="priority">
          <Input addonAfter="out of 99" max={99} min={1} type="number" />
        </Form.Item>
        <Form.Item
          extra="Priority is a whole number from 1 to 99 with 1 being the lowest priority."
          hidden={schedulerType !== api.V1SchedulerType.KUBERNETES}
          label="Priority"
          name="priority">
          <Input max={99} min={1} type="number" />
        </Form.Item>
        <Form.Item
          hidden={!process.env.IS_DEV || schedulerType === api.V1SchedulerType.FAIRSHARE}
          label="Position in Queue"
          name="position">
          <Input
            addonAfter={`out of ${jobs.length}`}
            max={jobs.length}
            min={1}
            type="number"
          />
        </Form.Item>
        <Form.Item
          hidden={schedulerType !== api.V1SchedulerType.FAIRSHARE}
          label="Weight"
          name="weight">
          <Input min={0} type="number" />
        </Form.Item>
        <Form.Item
          extra={poolDetails}
          label="Resource Pool"
          name="resourcePool">
          <Select disabled={job.type !== JobType.EXPERIMENT}>
            {resourcePools.map(rp => (
              <Option key={rp.name} value={rp.name}>{rp.name}</Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
      <h6>
        Job Details
      </h6>
      <List
        dataSource={details}
        renderItem={item => (
          <List.Item className={css.item}>
            <Typography.Text className={css.key}>{item.label}</Typography.Text>
            <div className={css.value}>
              {item.value}
            </div>
          </List.Item>
        )}
        size="small"
      />
    </Modal>
  );
};

export default ManageJob;
