import { Spin } from 'antd';
import React from 'react';

import Icon, { IconSize } from 'components/kit/Icon';

import css from './Spinner.module.scss';

interface Props {
  center?: boolean;
  children?: React.ReactNode;
  conditionalRender?: boolean;
  size?: IconSize;
  spinning: boolean;
  tip?: React.ReactNode;
}

const Spinner: React.FC<Props> = ({
  center,
  children,
  conditionalRender,
  size,
  spinning,
  tip,
}: Props) => {
  const classes = [css.base];

  if (center || tip) classes.push(css.center);

  return (
    <div className={classes.join(' ')}>
      <Spin
        data-testid="custom-spinner"
        indicator={
          <div className={css.spin}>
            <Icon name="spinner" size={size} title="Spinner" />
          </div>
        }
        spinning={spinning}
        tip={tip}>
        {conditionalRender ? (spinning ? null : children) : children}
      </Spin>
    </div>
  );
};

export default Spinner;
