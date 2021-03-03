import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import dayjs, { Dayjs } from 'dayjs';
import React from 'react';

import DatePickerFilter from 'components/DatePickerFilter';
import ResponsiveFilters from 'components/ResponsiveFilters';
import SelectFilter from 'components/SelectFilter';
import {
  DEFAULT_RANGE_DAY, DEFAULT_RANGE_MONTH, MAX_RANGE_DAY, MAX_RANGE_MONTH,
} from 'pages/Cluster/ClusterHistoricalUsage';
import { capitalize } from 'utils/string';

const { Option } = Select;

enum GroupBy {
  Day = 'day',
  Month = 'month',
}

export interface ClusterHistoricalUsageFiltersInterface {
  afterDate: Dayjs,
  beforeDate: Dayjs,
  groupBy: GroupBy,
}

interface ClusterHistoricalUsageFiltersProps {
  onChange: (newFilters: ClusterHistoricalUsageFiltersInterface) => void;
  value: ClusterHistoricalUsageFiltersInterface;
}

const ClusterHistoricalUsageFilters: React.FC<ClusterHistoricalUsageFiltersProps> = (
  { onChange, value }: ClusterHistoricalUsageFiltersProps,
) => {
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

  const handleAfterDateSelect = (afterDate: Dayjs|null) => {
    if (!afterDate) return;

    const dateDiff = value.beforeDate.diff(afterDate, value.groupBy);

    if (value.groupBy === GroupBy.Day && dateDiff >= MAX_RANGE_DAY) {
      value.beforeDate = afterDate.clone().add(MAX_RANGE_DAY - 1, 'day');
    }
    if (value.groupBy === GroupBy.Month && dateDiff >= MAX_RANGE_MONTH) {
      value.beforeDate = afterDate.clone().add(MAX_RANGE_MONTH - 1, 'month');
    }

    onChange({ ...value, afterDate });
  };

  const handleBeforeDateSelect = (beforeDate: Dayjs|null) => {
    if (!beforeDate) return;

    const dateDiff = beforeDate.diff(value.afterDate, value.groupBy);

    if (value.groupBy === GroupBy.Day && dateDiff >= MAX_RANGE_DAY) {
      value.afterDate = beforeDate.clone().subtract(MAX_RANGE_DAY - 1, 'day');
    }
    if (value.groupBy === GroupBy.Month && dateDiff >= MAX_RANGE_MONTH) {
      value.afterDate = beforeDate.clone().subtract(MAX_RANGE_MONTH - 1, 'month');
    }

    onChange({ ...value, beforeDate });
  };

  const isAfterDateDisabled = (currentDate: Dayjs) => {
    return currentDate.isAfter(value.beforeDate);
  };

  const isBeforeDateDisabled = (currentDate: Dayjs) => {
    return currentDate.isBefore(value.afterDate) || currentDate.isAfter(dayjs());
  };

  let periodFilters = null;
  if (value.groupBy === GroupBy.Day) {
    periodFilters = (
      <>
        <DatePickerFilter
          allowClear={false}
          disabledDate={isAfterDateDisabled}
          label='From'
          style={{ width: 130 }}
          value={value.afterDate}
          onChange={handleAfterDateSelect}
        />
        <DatePickerFilter
          allowClear={false}
          disabledDate={isBeforeDateDisabled}
          label='To'
          style={{ width: 130 }}
          value={value.beforeDate}
          onChange={handleBeforeDateSelect}
        />
      </>
    );
  }
  if (value.groupBy === GroupBy.Month) {
    periodFilters = (
      <>
        <DatePickerFilter
          allowClear={false}
          disabledDate={isAfterDateDisabled}
          label='From'
          picker='month'
          style={{ width: 130 }}
          value={value.afterDate}
          onChange={handleAfterDateSelect}
        />
        <DatePickerFilter
          allowClear={false}
          disabledDate={isBeforeDateDisabled}
          label='To'
          picker='month'
          style={{ width: 130 }}
          value={value.beforeDate}
          onChange={handleBeforeDateSelect}
        />
      </>
    );
  }

  return (
    <ResponsiveFilters>
      {periodFilters}
      <SelectFilter
        enableSearchFilter={false}
        label="Group by"
        showSearch={false}
        style={{ width: 130 }}
        value={value.groupBy}
        onSelect={handleGroupBySelect}>
        {Object.values(GroupBy).map(value => (
          <Option key={value} value={value}>{capitalize(value)}</Option>
        ))}
      </SelectFilter>
    </ResponsiveFilters>
  );
};

export default ClusterHistoricalUsageFilters;
