import { Spin } from 'antd';
import { SpinProps, SpinState } from 'antd/es/spin';
import React, { PropsWithChildren } from 'react';

import Icon from 'components/Icon';

import css from './Spinner.module.scss';

interface Props extends SpinProps, SpinState {}

export const IndicatorUnpositioned: React.FC = () => {
  const classes = [ css.spin ];
  return <div className={classes.join(' ')}>
    <Icon name="spinner" size="large" />
  </div>;
};

export const Indicator: React.FC = () => {
  const classes = [ css.spin, css.center ];
  return <div className={classes.join(' ')}>
    <Icon name="spinner" size="large" />
  </div>;
};

const Spinner: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return <Spin indicator={<Indicator />} {...props}>{props.children}</Spin>;
};

export default Spinner;
