import dayjs, { Dayjs } from 'dayjs';
import DatePicker from 'hew/DatePicker';
import Form from 'hew/Form';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import React from 'react';

import { handlePath, serverAddress } from 'routes/utils';
import { ValueOf } from 'types';
import handleError from 'utils/error';

export const CSVGroupBy = {
  Allocations: '/resources/allocation/allocations-csv?',
  Workloads: '/resources/allocation/raw?',
} as const;

export type CSVGroupBy = ValueOf<typeof CSVGroupBy>;

interface Props {
  afterDate: Dayjs;
  beforeDate: Dayjs;
  groupBy: CSVGroupBy;
  onVisibleChange: (visible: boolean) => void;
}

const ClusterHistoricalUsageCsvModalComponent: React.FC<Props> = ({
  afterDate,
  beforeDate,
  groupBy,
  onVisibleChange,
}: Props) => {
  const [form] = Form.useForm();

  const handleOk = (event: React.MouseEvent): void => {
    const formAfterDate = form.getFieldValue('afterDate');
    const formBeforeDate = form.getFieldValue('beforeDate');
    const groupByEndpoint = form.getFieldValue('groupBy');
    const searchParams = new URLSearchParams();

    searchParams.append('timestamp_after', formAfterDate.startOf('day').toISOString());
    searchParams.append('timestamp_before', formBeforeDate.endOf('day').toISOString());
    handlePath(event, {
      external: true,
      path: serverAddress(groupByEndpoint + searchParams.toString()),
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

  return (
    <Modal
      size="small"
      submit={{
        handleError: handleError,
        handler: handleOk,
        text: 'Proceed to Download',
      }}
      title="Download Resource Usage Data in CSV"
      onClose={() => onVisibleChange(false)}>
      <Form form={form} initialValues={{ afterDate, beforeDate, groupBy }} labelCol={{ span: 8 }}>
        <Form.Item label="Start" name="afterDate">
          <DatePicker allowClear={false} disabledDate={isAfterDateDisabled} width={150} />
        </Form.Item>
        <Form.Item label="End" name="beforeDate">
          <DatePicker allowClear={false} disabledDate={isBeforeDateDisabled} width={150} />
        </Form.Item>
        <Form.Item label="Group by" name="groupBy">
          <Select searchable={false} width={'150px'}>
            <Option value={CSVGroupBy.Workloads}>Workloads</Option>
            <Option value={CSVGroupBy.Allocations}>Allocations</Option>
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ClusterHistoricalUsageCsvModalComponent;
