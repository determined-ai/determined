import { DatePicker } from 'antd';
import { PickerProps } from 'antd/es/date-picker/generatePicker';
import { Dayjs } from 'dayjs';
import React from 'react';

import Label from './Label';
import css from './SelectFilter.module.scss';

type Props = PickerProps<Dayjs> & {
  tag: string;
}

const DatePickerFilter: React.FC<Props> = ({ tag, ...props }: Props) => {
  return (
    <div className={css.base}>
      <Label>{tag}</Label>
      <DatePicker {...props} />
    </div>
  );
};

export default DatePickerFilter;
