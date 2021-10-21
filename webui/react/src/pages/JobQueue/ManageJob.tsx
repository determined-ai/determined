import { Form, FormInstance, Input, Modal } from 'antd';
import React, { useCallback, useRef } from 'react';

import Json from 'components/Json';
import { detApi } from 'services/apiConfig';
import { Job } from 'types';
import { truncate } from 'utils/string';

interface Props {
  job: Job;
  onFinish: () => void;
}

const ManageJob: React.FC<Props> = ({ onFinish, job }) => {
  const formRef = useRef<FormInstance>(null);

  const onOk = useCallback(
    async () => {
      if (!formRef.current) return;
      const formValues = formRef.current.getFieldsValue();
      try {
        await detApi.Jobs.determinedUpdateJobQueue({
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
        console.error(
          'https://github.com/determined-ai/determined/pull/3039#issuecomment-947265893',
          e,
        );
      }
      onFinish();
    },
    [ formRef, onFinish, job.jobId, job.resourcePool ],
  );

  return (
    <Modal
      mask
      // style={{ minWidth: '600px' }}
      title={'Manage Job ' + truncate(job.jobId, 6, '')}
      visible={true}
      onCancel={onFinish}
      onOk={onOk}
    >
      <p>There are {job.summary.jobsAhead} jobs ahead of this job</p>
      <Form
        // className={css.form}
        // hidden={state.isAdvancedMode}
        initialValues={{
          priority: job.priority,
          queuePosition: -1,
          weight: job.weight,
        }}
        labelCol={{ span: 6 }}
        name="form basic"
        ref={formRef}
      >
        <Form.Item
          label="Q Position (DEV)"
          name="queuePosition"
          // rules={[ { message: 'Please provide a max length.', required: true } ]}
        >
          <Input disabled type="number" />
        </Form.Item>
        <Form.Item
          label="Priority"
          name="priority"
          // rules={[ { message: 'Please provide a new experiment name.', required: true } ]}
        >
          <Input type="number" />
        </Form.Item>
        <Form.Item
          label="Weight"
          name="weight"
          // rules={[ { message: 'Please provide a new experiment name.', required: true } ]}
        >
          <Input disabled type="number" />
        </Form.Item>
      </Form>
      <Json json={job} />
    </Modal>
  );
};

export default ManageJob;
