import { Form, FormInstance, Input, List, Modal, Typography } from 'antd';
import React, { ReactNode, useCallback, useMemo, useRef } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import handleError, { ErrorType } from 'ErrorHandler';
import { columns } from 'pages/JobQueue/JobQueue.table';
import * as api from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { Job, RPStats } from 'types';
import { moveJobToPositionUpdate, orderedSchedulers } from 'utils/job';
import { floatToPercent, truncate } from 'utils/string';

import css from './ManageJob.module.scss';

interface Props {
  job: Job;
  jobs: Job[];
  onFinish?: () => void;
  schedulerType: api.V1SchedulerType;
  selectedRPStats: RPStats;
}

interface FormValues {
  position?: string;
  priority?: string;
  resourcePool?: string;
  weight?: string;
}

const ManageJob: React.FC<Props> = ({ onFinish, selectedRPStats, job, schedulerType }) => {
  const formRef = useRef <FormInstance<FormValues>>(null);
  const isOrderedQ = orderedSchedulers.has(schedulerType);

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

  const formValuesToQUpdate = useCallback((
    formRef: React.RefObject<FormInstance<FormValues>>,
  ): api.V1QueueControl | undefined => {
    if (!formRef.current) return;
    const formValues = formRef.current.getFieldsValue();
    if (formValues.position !== undefined
      && parseInt(formValues.position, 10) - 1 !== job.summary.jobsAhead) {
      return moveJobToPositionUpdate(job.jobId, parseInt(formValues.position, 10));
    } else if (formValues.priority !== undefined
      && parseInt(formValues.priority) !== job.priority) {
      return { jobId: job.jobId, priority: parseInt(formValues.priority) };
    } else if (formValues.weight !== undefined && parseFloat(formValues.weight) !== job.weight) {
      return { jobId: job.jobId, weight: parseFloat(formValues.weight) };
    }
  }, [ job ]);

  const onOk = useCallback(
    async () => {
      try{
        const update = formValuesToQUpdate(formRef);
        if (update) await detApi.Internal.updateJobQueue({ updates: [ update ] });
      } catch (e) {
        handleError({
          error: e as Error,
          isUserTriggered: true,
          message: 'Failed to update job queue',
          publicMessage: `Failed to update job ${job.jobId}`,
          type: ErrorType.Api,
        });
      }
      onFinish?.();
    },
    [ formRef, onFinish, formValuesToQUpdate, job.jobId ],
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
      <Form
        initialValues={{
          position: job.summary.jobsAhead + 1,
          priority: job.priority,
          resourcePool: selectedRPStats.resourcePool,
          weight: job.weight,
        }}
        labelCol={{ span: 6 }}
        name="form basic"
        ref={formRef}>
        {schedulerType === api.V1SchedulerType.PRIORITY && (
          <Form.Item
            extra="Priority is a whole number from 1 to 99 with 1 being the highest priority."
            label="Priority"
            name="priority">
            <Input addonAfter="out of 99" max={99} min={1} type="number" />
          </Form.Item>
        )}
        {schedulerType === api.V1SchedulerType.KUBERNETES && (
          <Form.Item
            extra="Priority is a whole number from 1 to 99 with 1 being the lowest priority."
            label="Priority"
            name="priority">
            <Input max={99} min={1} type="number" />
          </Form.Item>
        )}
        {schedulerType === api.V1SchedulerType.FAIRSHARE && (
          <Form.Item
            label="Weight"
            name="weight">
            <Input min={0} type="number" />
          </Form.Item>
        )}
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
