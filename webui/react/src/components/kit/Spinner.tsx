import { Spin } from 'antd';
import React from 'react';

import Icon, { IconSize } from 'components/kit/Icon';
import { Loadable } from 'utils/loadable';

import css from './Spinner.module.scss';

interface PropsBase {
  center?: boolean;
  size?: IconSize;
  tip?: React.ReactNode;
}

type Props<T> = (
  | {
      children?: React.ReactNode;
      conditionalRender?: boolean;
      data?: never;
      spinning: boolean;
    }
  | {
      children: (data: T) => JSX.Element;
      conditionalRender?: never;
      data: Loadable<T>;
      spinning?: never;
    }
) &
  PropsBase;

function Spinner<T>({
  center,
  children,
  conditionalRender,
  size = 'medium',
  spinning,
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
        spinning={spinning ?? true}
        tip={tip}>
        {spinning === undefined || (conditionalRender && spinning) ? null : children}
      </Spin>
    </div>
  );

  if (spinning !== undefined) {
    return spinner;
  } else {
    return Loadable.match(data, {
      Loaded: children,
      NotLoaded: () => spinner,
    });
  }
}

export default Spinner;
