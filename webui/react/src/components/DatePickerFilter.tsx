import { DatePicker } from 'antd';
import { Moment } from 'moment';
import React from 'react';

import css from './DatePickerFilter.module.scss';
import Label from './Label';

interface Props {
  label: string;
  onChange?: (date: Moment | null) => void;
  value?: Moment;
}

const DatePickerFilter: React.FC<Props> = ({ label, onChange, value }: Props) => {
  return (
    <div className={css.wrapper}>
      <Label>{label}</Label>
      <DatePicker
        allowClear={false}
        showNow={false}
        showTime
        style={{ width: '130px' }}
        value={value}
        onChange={onChange}
      />
    </div>
  );
};

export default DatePickerFilter;
