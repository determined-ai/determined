import { DatePicker as AntdDatePicker } from 'antd';
import { PickerProps } from 'antd/es/date-picker/generatePicker';
import { Dayjs } from 'dayjs';
import React from 'react';

import css from 'components/DatePicker.module.scss';
import Label from 'components/Label';

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
