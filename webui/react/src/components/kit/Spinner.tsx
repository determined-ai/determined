import { Spin } from 'antd';
import React from 'react';

import Icon, { IconSize } from 'components/kit/Icon';
import { XOR } from 'components/kit/internal/types';
import { Loadable } from 'components/kit/utils/loadable';

import css from './Spinner.module.scss';

interface PropsBase {
  center?: boolean;
  size?: IconSize;
  tip?: React.ReactNode;
}

type Props<T> = XOR<
  {
    children?: React.ReactNode;
    conditionalRender?: boolean;
    spinning?: boolean;
  },
  {
    children: (data: T) => JSX.Element;
    data: Loadable<T>;
  }
> &
  PropsBase;

function Spinner<T>({
  center,
  children,
  conditionalRender,
  size = 'medium',
  spinning = true,
  tip,
  data,
}: Props<T>): JSX.Element {
  const classes = [css.base];

  if (center || tip) classes.push(css.center);

  const spinner = (
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
        {data !== undefined || (conditionalRender && spinning) ? null : children}
      </Spin>
    </div>
  );

  if (!data) {
    return spinner;
  } else {
    return Loadable.match(data, {
      Failed: () => <></>,
      Loaded: children,
      NotLoaded: () => spinner, // TODO circle back with design to find an appropriate error state here
    });
  }
}

export default Spinner;
