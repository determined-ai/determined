import { Form, FormInstance, Input, Modal, Select } from 'antd';
import React, { useCallback, useRef } from 'react';

import Json from 'components/Json';
import { useStore } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { V1SchedulerType } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { Job, RPStats } from 'types';
import { truncate } from 'utils/string';

import RPStatsOverview from './RPStats';

const { Option } = Select;

interface Props {
  job: Job;
  onFinish?: () => void;
  schedulerType: V1SchedulerType;
  selectedRPStats: RPStats;
}

const ManageJob: React.FC<Props> = ({ onFinish, selectedRPStats, job, schedulerType }) => {
  const formRef = useRef<FormInstance>(null);
  const { resourcePools } = useStore();

  const onOk = useCallback(
    async () => {
      if (!formRef.current) return;
      const formValues = formRef.current.getFieldsValue();
      try {
        await detApi.Internal.updateJobQueue({
          updates: [
            // TODO validate & avoid including all 3
            {
              jobId: job.jobId,
              priority: parseInt(formValues.priority),
              // queuePosition: parseFloat(formValues.queuePosition),
              sourceResourcePool: job.resourcePool,
              // weight: parseInt(formValues.weight),
            },
          ],
        });
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
    [ formRef, onFinish, job.jobId, job.resourcePool ],
  );

  const curRP = resourcePools.find(rp => rp.name === selectedRPStats.resourcePool);

  const RPDetails = (
    <div>
      <p>Current slot allocation: {curRP?.slotsUsed} / {curRP?.slotsAvailable} </p>
      <RPStatsOverview stats={selectedRPStats} />
      <p>Jobs in queue:
        {selectedRPStats.stats.queuedCount + selectedRPStats.stats.scheduledCount}
      </p>
      <p>Spot instance pool: {!!curRP?.details.aws?.spotEnabled + ''}</p>
    </div>
  );

  const isSingular = job.summary && job.summary.jobsAhead === 1;

  return (
    <Modal
      mask
      // style={{ minWidth: '600px' }}
      title={'Manage Job ' + truncate(job.jobId, 6, '')}
      visible={true}
      onCancel={onFinish}
      onOk={onOk}>
      <p>There {isSingular ? 'is' : 'are'} {job.summary?.jobsAhead || 'No'} job
        {isSingular ? '' : 's'} ahead of this job.
      </p>
      <Form
        // className={css.form}
        // hidden={state.isAdvancedMode}
        initialValues={{
          priority: job.priority,
          queuePosition: -1,
          resourcePool: selectedRPStats.resourcePool,
          weight: job.weight,
        }}
        labelCol={{ span: 6 }}
        name="form basic"
        ref={formRef}>
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
        <Form.Item
          label="Q Position (DEV)"
          name="queuePosition">
          <Input disabled type="number" />
        </Form.Item>
        {schedulerType === V1SchedulerType.PRIORITY && (
          <Form.Item
            extra="Jobs are scheduled based on priority of 1-99 with 1 being the highest priority."
            label="Priority"
            name="priority">
            <Input addonAfter="out of 99" type="number" />
          </Form.Item>
        )}
        {schedulerType === V1SchedulerType.FAIRSHARE && (
          <Form.Item
            label="Weight"
            name="weight">
            <Input disabled type="number" />
          </Form.Item>
        )}
      </Form>
      <Json json={job} />
    </Modal>
  );
};

export default ManageJob;
