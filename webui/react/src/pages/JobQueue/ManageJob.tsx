import Form from 'hew/Form';
import Glossary from 'hew/Glossary';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import { Loadable } from 'hew/utils/loadable';
import React, { ReactElement, ReactNode, useCallback, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import useFeature from 'hooks/useFeature';
import { columns } from 'pages/JobQueue/JobQueue.table';
import { updateJobQueue } from 'services/api';
import * as api from 'services/api-ts-sdk';
import clusterStore from 'stores/cluster';
import { Job, JobType } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { orderedSchedulers } from 'utils/job';
import { useObservable } from 'utils/observable';
import { floatToPercent, truncate } from 'utils/string';

interface Props {
  initialPool: string;
  job: Job;
  onFinish?: () => void;
  rpStats: api.V1RPQueueStat[];
  schedulerType: api.V1SchedulerType;
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

const formValuesToUpdate = (values: FormValues, job: Job): api.V1QueueControl | undefined => {
  const { resourcePool } = {
    resourcePool: values.resourcePool,
  };
  const update: api.V1QueueControl = { jobId: job.jobId };

  if (resourcePool !== job.resourcePool) {
    return { ...update, resourcePool };
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
  return undefined;
};

const ManageJobModalComponent: React.FC<Props> = ({
  onFinish,
  rpStats,
  job,
  schedulerType,
  initialPool,
}) => {
  const [form] = Form.useForm();
  const isOrderedQ = orderedSchedulers.has(schedulerType);
  const resourcePools = Loadable.getOrElse([], useObservable(clusterStore.resourcePools)); // TODO show spinner when this is loading
  const [selectedPoolName, setSelectedPoolName] = useState(initialPool);
  const f_flat_runs = useFeature().isOn('flat_runs');

  const details = useMemo(() => {
    interface Item {
      label: ReactNode;
      value: ReactNode;
    }
    const tableKeys = ['user', 'slots', 'submitted', 'type', 'name'];
    const tableDetails: Record<string, Item> = {};

    tableKeys.forEach((td) => {
      const col = columns(f_flat_runs).find((col) => col.key === td);
      if (!col?.render) return;
      tableDetails[td] = { label: <>{col.title}</>, value: <>{col.render(undefined, job, 0)}</> };
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
        value: job.progress ? floatToPercent(job.progress, 1) : undefined,
      },
      tableDetails.slots,
      { label: 'Is Preemtible', value: job.isPreemptible ? 'Yes' : 'No' },
      {
        label: 'Jobs Ahead',
        value: isOrderedQ ? job.summary.jobsAhead : undefined,
      },
      tableDetails.user,
    ];

    return items.filter((item) => !!item && item.value !== undefined) as Item[];
  }, [job, isOrderedQ, f_flat_runs]);

  const currentPool = useMemo(() => {
    return resourcePools.find((rp) => rp.name === selectedPoolName);
  }, [resourcePools, selectedPoolName]);

  const currentPoolStats = useMemo(() => {
    return rpStats.find((rp) => rp.resourcePool === selectedPoolName);
  }, [rpStats, selectedPoolName]);

  const poolDetails = useMemo(() => {
    return (
      <div>
        <p>
          Current slot allocation: {currentPool?.slotsUsed} / {currentPool?.slotsAvailable} (used /
          total)
          <br />
          Jobs in queue:
          {(currentPoolStats?.stats.queuedCount ?? 0) +
            (currentPoolStats?.stats.scheduledCount ?? 0)}
          <br />
          Spot instance pool: {!!currentPool?.details.aws?.spotEnabled + ''}
        </p>
      </div>
    );
  }, [currentPool, currentPoolStats]);

  // eslint-disable-next-line  @typescript-eslint/no-explicit-any
  const handleUpdateResourcePool = useCallback((changedValues: any) => {
    if (changedValues.resourcePool) setSelectedPoolName(changedValues.resourcePool);
  }, []);

  const onOk = useCallback(async () => {
    try {
      const update = form && (await formValuesToUpdate(form.getFieldsValue(), job));
      if (update) await updateJobQueue({ updates: [update] });
    } catch (e) {
      handleError(e, {
        isUserTriggered: true,
        publicSubject: 'Failed to update the job.',
        silent: false,
        type: ErrorType.Api,
      });
    }
    onFinish?.();
  }, [form, onFinish, job]);

  const isSingular = job.summary && job.summary.jobsAhead === 1;

  return (
    <Modal
      submit={{
        handleError: () => {},
        handler: onOk,
        text: 'Close',
      }}
      title={'Manage Job ' + truncate(job.jobId, 6, '')}
      onClose={onFinish}>
      {isOrderedQ && (
        <p>
          There {isSingular ? 'is' : 'are'} {job.summary?.jobsAhead || 'no'} job
          {isSingular ? '' : 's'} ahead of this job.
        </p>
      )}
      <h6>Queue Settings</h6>
      <Form
        form={form}
        initialValues={{
          position: job.summary.jobsAhead + 1,
          priority: job.priority,
          resourcePool: initialPool,
          weight: job.weight,
        }}
        labelCol={{ span: 6 }}
        name="form basic"
        onValuesChange={handleUpdateResourcePool}>
        <Form.Item
          extra="Priority is a whole number from 1 to 99 with 1 being the highest priority."
          hidden={schedulerType !== api.V1SchedulerType.PRIORITY}
          label="Priority"
          name="priority">
          <Input addonAfter="out of 99" max={99} min={1} type="number" />
        </Form.Item>
        <Form.Item
          extra="Priority is a whole number from 1 to 99 with 1 being the lowest priority.
          Adjusting the priority will cancel and resubmit the job to update its priority."
          hidden={schedulerType !== api.V1SchedulerType.KUBERNETES}
          label="Priority"
          name="priority">
          <Input max={99} min={1} type="number" />
        </Form.Item>
        <Form.Item
          hidden={schedulerType !== api.V1SchedulerType.FAIRSHARE}
          label="Weight"
          name="weight">
          <Input min={0} type="number" />
        </Form.Item>
        <Form.Item
          extra={poolDetails}
          hidden={schedulerType === api.V1SchedulerType.KUBERNETES}
          label="Resource Pool"
          name="resourcePool">
          <Select disabled={job.type !== JobType.EXPERIMENT}>
            {resourcePools.map((rp) => (
              <Option key={rp.name} value={rp.name}>
                {rp.name}
              </Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
      <h6>Job Details</h6>
      <Glossary
        content={details.map((item) => {
          return {
            label: item.label as string,
            value: item.value as ReactElement,
          };
        })}
      />
    </Modal>
  );
};

export default ManageJobModalComponent;
