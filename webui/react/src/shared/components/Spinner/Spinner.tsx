import { Spin } from 'antd';
import type { SpinProps } from 'antd/es/spin';
import React from 'react';

import Icon, { IconSize } from 'components/kit/Icon';

import css from './Spinner.module.scss';

interface Props extends Omit<SpinProps, 'size'> {
  center?: boolean;
  children?: React.ReactNode;
  conditionalRender?: boolean;
  inline?: boolean;
  size?: IconSize;
}

const Spinner: React.FC<Props> = ({
  center,
  className,
  conditionalRender,
  size,
  spinning,
  tip,
  ...props
}: Props) => {
  const classes = [css.base];

  if (className) classes.push(className);
  if (center || tip) classes.push(css.center);

  return (
    <div className={classes.join(' ')}>
      <Spin
        data-testid="custom-spinner"
        indicator={
          <div className={css.spin}>
            <Icon name="spinner" size={size} />
          </div>
        }
        spinning={!!spinning}
        tip={tip}
        {...props}>
        {conditionalRender ? (spinning ? null : props.children) : props.children}
      </Spin>
    </div>
  );
};

export default Spinner;
