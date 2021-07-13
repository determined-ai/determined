import { Spin } from 'antd';
import { SpinProps, SpinState } from 'antd/es/spin';
import React, { PropsWithChildren } from 'react';

import Icon, { IconSize } from 'components/Icon';

import css from './Spinner.module.scss';

interface Props extends SpinProps, SpinState {}

interface IndicatorUnpositionedProps {
  size?: IconSize;
}

export const IndicatorUnpositioned: React.FC<IndicatorUnpositionedProps> =(
  { size = 'large' }: IndicatorUnpositionedProps,
) => {
  const classes = [ css.spin ];
  return <div className={classes.join(' ')}>
    <Icon name="spinner" size={size} />
  </div>;
};

export const Indicator: React.FC = () => {
  const classes = [ css.spin, css.center ];
  return <div className={classes.join(' ')}>
    <Icon name="spinner" size="large" />
  </div>;
};

const Spinner: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return <div className={css.centerContainer}>
    <Spin
      indicator={props.tip ? <IndicatorUnpositioned /> : <Indicator />}
      {...props}>{props.children}</Spin>
  </div>;
};

export default Spinner;
