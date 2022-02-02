import { Spin } from 'antd';
import { SpinProps, SpinState } from 'antd/es/spin';
import React, { PropsWithChildren } from 'react';

import Icon, { IconSize } from 'components/Icon';

import css from './Spinner.module.scss';

interface Props extends Omit<SpinProps, 'size'>, SpinState {
  center?: boolean;
  inline?: boolean;
  size?: IconSize;
}

const Spinner: React.FC<Props> = ({
  center,
  className,
  size,
  tip,
  ...props
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];

  if (className) classes.push(className);
  if (center || tip) classes.push(css.center);

  return (
    <div className={classes.join(' ')}>
      <Spin
        indicator={(
          <div className={css.spin}>
            <Icon name="spinner" size={size} />
          </div>
        )}
        tip={tip}
        {...props}>
        {props.children}
      </Spin>
    </div>
  );
};

export default Spinner;
