import { DatePicker, Form, Modal } from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import React from 'react';

import { handlePath, serverAddress } from 'routes/utils';

interface Props {
  afterDate: Dayjs,
  beforeDate: Dayjs,
  onVisibleChange: (visible: boolean) => void;
}

const ClusterHistoricalUsageCsvModal: React.FC<Props> = (
  { afterDate, beforeDate, onVisibleChange }: Props,
) => {
  const [ form ] = Form.useForm();

  const handleOk = (event: React.MouseEvent): void => {
    const formAfterDate = form.getFieldValue('afterDate');
    const formBeforeDate = form.getFieldValue('beforeDate');
    const searchParams = new URLSearchParams;

    searchParams.append('timestamp_after', formAfterDate.startOf('day').toISOString());
    searchParams.append('timestamp_before', formBeforeDate.endOf('day').toISOString());

    handlePath(event, {
      external: true,
      path: serverAddress('/resources/allocation/raw?' + searchParams.toString()),
      popout: true,
    });

    onVisibleChange(false);
  };

  const isAfterDateDisabled = (currentDate: Dayjs) => {
    const formBeforeDate = form.getFieldValue('beforeDate');
    return currentDate.isAfter(formBeforeDate);
  };

  const isBeforeDateDisabled = (currentDate: Dayjs) => {
    const formAfterDate = form.getFieldValue('afterDate');
    return currentDate.isBefore(formAfterDate) || currentDate.isAfter(dayjs());
  };

  return <Modal
    okText='Proceed to Download'
    title='Download Resource Usage Data in CSV'
    visible={true}
    onCancel={() => onVisibleChange(false)}
    onOk={handleOk}
  >
    <Form
      form={form}
      initialValues={{ afterDate, beforeDate }}
      labelCol={{ span: 8 }}
    >
      <Form.Item label="Start" name="afterDate">
        <DatePicker
          allowClear={false}
          disabledDate={isAfterDateDisabled}
          style={{ minWidth: '150px' }}
        />
      </Form.Item>

      <Form.Item label="End" name="beforeDate">
        <DatePicker
          allowClear={false}
          disabledDate={isBeforeDateDisabled}
          style={{ minWidth: '150px' }}
        />
      </Form.Item>
    </Form>
  </Modal>;
};

export default ClusterHistoricalUsageCsvModal;
