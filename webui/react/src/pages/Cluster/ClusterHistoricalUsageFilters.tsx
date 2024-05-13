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
  afterDate: Dayjs;
  beforeDate: Dayjs;
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
        afterDate: dayjs().subtract(DEFAULT_RANGE_MONTH, 'month'),
        beforeDate: dayjs(),
        groupBy: GroupBy.Month,
      });
    } else {
      onChange({
        afterDate: dayjs().subtract(1 + DEFAULT_RANGE_DAY, 'day'),
        beforeDate: dayjs().subtract(1, 'day'),
        groupBy: GroupBy.Day,
      });
    }
  };

  const handleAfterDateSelect = (afterDate: Dayjs | null) => {
    if (!afterDate) return;
    const val = _.cloneDeep(value);

    const dateDiff = val.beforeDate.diff(afterDate, val.groupBy);

    if (val.groupBy === GroupBy.Day && dateDiff >= MAX_RANGE_DAY) {
      val.beforeDate = afterDate.clone().add(MAX_RANGE_DAY - 1, 'day');
    }
    if (val.groupBy === GroupBy.Month && dateDiff >= MAX_RANGE_MONTH) {
      val.beforeDate = afterDate.clone().add(MAX_RANGE_MONTH - 1, 'month');
    }

    onChange({ ...val, afterDate });
  };

  const handleBeforeDateSelect = (beforeDate: Dayjs | null) => {
    if (!beforeDate) return;
    const val = _.cloneDeep(value);

    const dateDiff = beforeDate.diff(val.afterDate, val.groupBy);

    if (val.groupBy === GroupBy.Day && dateDiff >= MAX_RANGE_DAY) {
      val.afterDate = beforeDate.clone().subtract(MAX_RANGE_DAY - 1, 'day');
    }
    if (val.groupBy === GroupBy.Month && dateDiff >= MAX_RANGE_MONTH) {
      val.afterDate = beforeDate.clone().subtract(MAX_RANGE_MONTH - 1, 'month');
    }

    onChange({ ...val, beforeDate });
  };

  const isAfterDateDisabled = (currentDate: Dayjs) => {
    return currentDate.isAfter(value.beforeDate);
  };

  const isBeforeDateDisabled = (currentDate: Dayjs) => {
    return currentDate.isBefore(value.afterDate) || currentDate.isAfter(dayjs());
  };

  let periodFilters: React.ReactNode = undefined;
  if (value.groupBy === GroupBy.Day) {
    periodFilters = (
      <>
        <DatePicker
          allowClear={false}
          disabledDate={isAfterDateDisabled}
          label="From"
          value={value.afterDate}
          width={130}
          onChange={handleAfterDateSelect}
        />
        <DatePicker
          allowClear={false}
          disabledDate={isBeforeDateDisabled}
          label="To"
          value={value.beforeDate}
          width={130}
          onChange={handleBeforeDateSelect}
        />
      </>
    );
  }
  if (value.groupBy === GroupBy.Month) {
    periodFilters = (
      <>
        <DatePicker
          allowClear={false}
          disabledDate={isAfterDateDisabled}
          label="From"
          picker="month"
          value={value.afterDate}
          width={130}
          onChange={handleAfterDateSelect}
        />
        <DatePicker
          allowClear={false}
          disabledDate={isBeforeDateDisabled}
          label="To"
          picker="month"
          value={value.beforeDate}
          width={130}
          onChange={handleBeforeDateSelect}
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
