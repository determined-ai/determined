import { DatePicker } from 'antd';
import { PickerProps } from 'antd/es/date-picker/generatePicker';
import { Dayjs } from 'dayjs';
import React from 'react';

import Label from './Label';
import css from './SelectFilter.module.scss';

type Props = PickerProps<Dayjs> & {
  label: string;
}

const DatePickerFilter: React.FC<Props> = ({ label, ...props }: Props) => {
  return (
    <div className={css.base}>
      <Label>{label}</Label>
      <DatePicker {...props} />
    </div>
  );
};

export default DatePickerFilter;
