import { DatePicker } from 'antd';
import { Dayjs } from 'dayjs';
import React from 'react';

import Label from './Label';
import css from './SelectFilter.module.scss';

interface Props {
  className?: string;
  label: string;
  onChange?: (date: Dayjs | null) => void;
  value?: Dayjs;
}

const DatePickerFilter: React.FC<Props> = ({ className = '', label, onChange, value }: Props) => {
  const classes = [ className, css.base ];

  return (
    <div className={classes.join(' ')}>
      <Label>{label}</Label>
      <DatePicker
        showNow={false}
        showTime
        value={value}
        onChange={onChange}
      />
    </div>
  );
};

export default DatePickerFilter;
