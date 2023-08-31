import { DatePicker as AntdDatePicker } from 'antd';
import { PickerProps } from 'antd/es/date-picker/generatePicker';
import { Dayjs } from 'dayjs';
import React from 'react';

import Label from 'components/Label';

import css from './DatePicker.module.scss';

type Props = PickerProps<Dayjs> & {
  label: string;
};

const DatePicker: React.FC<Props> = ({ label, ...props }: Props) => {
  return (
    <div className={css.base}>
      <Label>{label}</Label>
      <AntdDatePicker {...props} />
    </div>
  );
};

export default DatePicker;
