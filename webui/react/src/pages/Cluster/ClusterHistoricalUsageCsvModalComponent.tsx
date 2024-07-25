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
  fromDate: Dayjs;
  toDate: Dayjs;
  groupBy: CSVGroupBy;
  onVisibleChange: (visible: boolean) => void;
}

const ClusterHistoricalUsageCsvModalComponent: React.FC<Props> = ({
  fromDate,
  toDate,
  groupBy,
  onVisibleChange,
}: Props) => {
  const [form] = Form.useForm();

  const handleOk = (event: React.MouseEvent): void => {
    const formFromDate = form.getFieldValue('fromDate');
    const formToDate = form.getFieldValue('toDate');
    const groupByEndpoint = form.getFieldValue('groupBy');
    const searchParams = new URLSearchParams();

    searchParams.append('timestamp_after', formFromDate.startOf('day').toISOString());
    searchParams.append('timestamp_before', formToDate.endOf('day').toISOString());
    handlePath(event, {
      external: true,
      path: serverAddress(groupByEndpoint + searchParams.toString()),
    });
    onVisibleChange(false);
  };

  const isfromDateDisabled = (currentDate: Dayjs) => {
    const formtoDate = form.getFieldValue('toDate');
    return currentDate.isAfter(formtoDate);
  };

  const istoDateDisabled = (currentDate: Dayjs) => {
    const formfromDate = form.getFieldValue('fromDate');
    return currentDate.isBefore(formfromDate) || currentDate.isAfter(dayjs());
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
      <Form form={form} initialValues={{ fromDate, groupBy, toDate }} labelCol={{ span: 8 }}>
        <Form.Item label="Start" name="fromDate">
          <DatePicker allowClear={false} disabledDate={isfromDateDisabled} width={150} />
        </Form.Item>
        <Form.Item label="End" name="toDate">
          <DatePicker allowClear={false} disabledDate={istoDateDisabled} width={150} />
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
