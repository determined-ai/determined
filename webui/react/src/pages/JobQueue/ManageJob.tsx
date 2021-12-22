import { Form, FormInstance, Input, Modal, Select } from 'antd';
import React, { useCallback, useRef } from 'react';

import Json from 'components/Json';
import { useStore } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { V1SchedulerType } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { Job, RPStats } from 'types';
import { truncate } from 'utils/string';

import { moveJobToPosition } from './utils';

const { Option } = Select;

interface Props {
  job: Job;
  jobs: Job[];
  onFinish?: () => void;
  schedulerType: V1SchedulerType;
  selectedRPStats: RPStats;
}

const ManageJob: React.FC<Props> = ({ onFinish, selectedRPStats, job, schedulerType, jobs }) => {
  const formRef = useRef<FormInstance>(null);
  const { resourcePools } = useStore();

  const onOk = useCallback(
    async () => {
      if (!formRef.current) return;
      const formValues = formRef.current.getFieldsValue();
      try {
        // TODO better detection?
        const jobsAhead = parseInt(formValues.position, 10) - 1;
        if (jobsAhead !== job.summary.jobsAhead) {
          await moveJobToPosition(job.jobId, jobsAhead);
        } else {
          await detApi.Internal.updateJobQueue({
            updates: [
            // TODO validate & avoid including all 3
              {
                jobId: job.jobId,
                priority: parseInt(formValues.priority),
                weight: parseFloat(formValues.weight),
              },
            ],
          });
        }

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
    [ formRef, onFinish, job.jobId, job.summary.jobsAhead ],
  );

  const curRP = resourcePools.find(rp => rp.name === selectedRPStats.resourcePool);

  const RPDetails = (
    <div>
      <p>Current slot allocation: {curRP?.slotsUsed} / {curRP?.slotsAvailable}
        <br />
        Jobs in queue:
        {selectedRPStats.stats.queuedCount + selectedRPStats.stats.scheduledCount}
        <br />
        Spot instance pool: {!!curRP?.details.aws?.spotEnabled + ''}
      </p>
    </div>
  );

  const isSingular = job.summary && job.summary.jobsAhead === 1;

  return (
    <Modal
      mask
      title={'Manage Job ' + truncate(job.jobId, 6, '')}
      visible={true}
      onCancel={onFinish}
      onOk={onOk}>
      <p>There {isSingular ? 'is' : 'are'} {job.summary?.jobsAhead || 'no'} job
        {isSingular ? '' : 's'} ahead of this job.
      </p>
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
        {schedulerType === V1SchedulerType.PRIORITY && (
          <>
            <Form.Item
              label="Position in Queue"
              name="position">
              <Input addonAfter={`out of ${jobs.length}`} max={jobs.length} min={1} type="number" />
            </Form.Item>
            <Form.Item
              // extra="Jobs are scheduled based on priority of 1-99 with 1 being the highest."
              label="Priority"
              name="priority">
              <Input addonAfter="out of 99" type="number" />
              {/* FIXME What about K8? */}
            </Form.Item>
          </>
        )}
        {schedulerType === V1SchedulerType.FAIRSHARE && (
          <Form.Item
            label="Weight"
            name="weight">
            <Input disabled type="number" />
          </Form.Item>
        )}
        <Form.Item
          extra={RPDetails}
          label="Resource Pool"
          name="resourcePool">
          <Select disabled>
            {resourcePools.map(rp => (
              <Option key={rp.name} value={rp.name}>{rp.name}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          extra="Change the resource request. Note: this can only be modified before a job is run."
          label="Slots"
          name="slots">
          <Input disabled type="number" />
        </Form.Item>
      </Form>
      <h6>
        Job Details
      </h6>
      <Json json={job} />
    </Modal>
  );
};

export default ManageJob;
