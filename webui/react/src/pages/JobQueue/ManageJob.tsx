import { Modal } from 'antd';
import React from 'react';

import { Job } from 'types';
import { truncate } from 'utils/string';

interface Props {
  job?: Job;
  onFinish: () => void;
}

const ManageJob: React.FC<Props> = ({ onFinish, job }) => {
  if (!job) return null;
  return (
    <Modal
      cancelButtonProps={{ style: { display: 'none' } }}
      cancelText=""
      mask
      // style={{ minWidth: '600px' }}
      title={'Manage Job ' + truncate(job.jobId, 6, '')}
      visible={true}
      onCancel={onFinish}
      onOk={onFinish}
    />
  );
};

export default ManageJob;
