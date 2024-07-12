import dayjs, { Dayjs } from 'dayjs';
import DatePicker from 'hew/DatePicker';
import Select, { Option, SelectValue } from 'hew/Select';
import _ from 'lodash';
import React from 'react';

import ResponsiveFilters from 'components/ResponsiveFilters';
import {
  DEFAULT_RANGE_DAY,
  DEFAULT_RANGE_MONTH,
  MAX_RANGE_DAY,
  MAX_RANGE_MONTH,
} from 'pages/Cluster/ClusterHistoricalUsage';
import { ValueOf } from 'types';
import { capitalize } from 'utils/string';

const GroupBy = {
  Day: 'day',
  Month: 'month',
} as const;

type GroupBy = ValueOf<typeof GroupBy>;

export interface ClusterHistoricalUsageFiltersInterface {
  fromDate: Dayjs;
  toDate: Dayjs;
  groupBy: GroupBy;
}

interface ClusterHistoricalUsageFiltersProps {
  onChange: (newFilters: ClusterHistoricalUsageFiltersInterface) => void;
  value: ClusterHistoricalUsageFiltersInterface;
}

const ClusterHistoricalUsageFilters: React.FC<ClusterHistoricalUsageFiltersProps> = ({
  onChange,
  value,
}: ClusterHistoricalUsageFiltersProps) => {
  const handleGroupBySelect = (groupBy: SelectValue) => {
    if (groupBy === GroupBy.Month) {
      onChange({
        fromDate: dayjs().subtract(DEFAULT_RANGE_MONTH, 'month'),
        groupBy: GroupBy.Month,
        toDate: dayjs(),
      });
    } else {
      onChange({
        fromDate: dayjs().subtract(1 + DEFAULT_RANGE_DAY, 'day'),
        groupBy: GroupBy.Day,
        toDate: dayjs().subtract(1, 'day'),
      });
    }
  };

  const handleFromDateSelect = (fromDate: Dayjs | null) => {
    if (!fromDate) return;
    const val = _.cloneDeep(value);

    const dateDiff = val.toDate.diff(fromDate, val.groupBy);

    if (val.groupBy === GroupBy.Day && dateDiff >= MAX_RANGE_DAY) {
      val.toDate = fromDate.clone().add(MAX_RANGE_DAY - 1, 'day');
    }
    if (val.groupBy === GroupBy.Month && dateDiff >= MAX_RANGE_MONTH) {
      val.toDate = fromDate.clone().add(MAX_RANGE_MONTH - 1, 'month');
    }

    onChange({ ...val, fromDate: fromDate });
  };

  const handleToDateSelect = (toDate: Dayjs | null) => {
    if (!toDate) return;
    const val = _.cloneDeep(value);

    const dateDiff = toDate.diff(val.fromDate, val.groupBy);

    if (val.groupBy === GroupBy.Day && dateDiff >= MAX_RANGE_DAY) {
      val.fromDate = toDate.clone().subtract(MAX_RANGE_DAY - 1, 'day');
    }
    if (val.groupBy === GroupBy.Month && dateDiff >= MAX_RANGE_MONTH) {
      val.fromDate = toDate.clone().subtract(MAX_RANGE_MONTH - 1, 'month');
    }

    onChange({ ...val, toDate });
  };

  const isFromDateDisabled = (currentDate: Dayjs) => {
    return currentDate.isAfter(value.toDate);
  };

  const isToDateDisabled = (currentDate: Dayjs) => {
    return currentDate.isBefore(value.fromDate) || currentDate.isAfter(dayjs());
  };

  let periodFilters: React.ReactNode = undefined;
  if (value.groupBy === GroupBy.Day) {
    periodFilters = (
      <>
        <DatePicker
          allowClear={false}
          disabledDate={isFromDateDisabled}
          label="From"
          value={value.fromDate}
          width={130}
          onChange={handleFromDateSelect}
        />
        <DatePicker
          allowClear={false}
          disabledDate={isToDateDisabled}
          label="To"
          value={value.toDate}
          width={130}
          onChange={handleToDateSelect}
        />
      </>
    );
  }
  if (value.groupBy === GroupBy.Month) {
    periodFilters = (
      <>
        <DatePicker
          allowClear={false}
          disabledDate={isFromDateDisabled}
          label="From"
          picker="month"
          value={value.fromDate}
          width={130}
          onChange={handleFromDateSelect}
        />
        <DatePicker
          allowClear={false}
          disabledDate={isToDateDisabled}
          label="To"
          picker="month"
          value={value.toDate}
          width={130}
          onChange={handleToDateSelect}
        />
      </>
    );
  }

  return (
    <ResponsiveFilters>
      {periodFilters}
      <Select
        label="Group by"
        searchable={false}
        value={value.groupBy}
        width={90}
        onSelect={handleGroupBySelect}>
        {Object.values(GroupBy).map((value) => (
          <Option key={value} value={value}>
            {capitalize(value)}
          </Option>
        ))}
      </Select>
    </ResponsiveFilters>
  );
};

export default ClusterHistoricalUsageFilters;
