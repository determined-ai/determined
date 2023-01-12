import { Spin } from 'antd';
import type { SpinProps } from 'antd/es/spin';
import React, { ReactElement, ReactNode } from 'react';

import Icon, { IconSize } from 'shared/components/Icon/Icon';
import { UnknownRecord } from 'shared/types';
import { Loadable, Loaded } from 'utils/loadable';

import css from './Spinner.module.scss';

// import { ReactNode } from 'react';
// import { Loadable } from 'utils/loadable';

// interface Props<T> {
//   loadables: { [K in keyof T]: Loadable<T[K]> };
//   children: ReactNode;
// }

// type LoadableSpinner<T> = React.FC<Props<T>>

// export const LoadingSpinner<T>: React.FC<Props<T>> = ({ loadables, children }) => {
//   return <></>;

// };

interface Props<T> extends Omit<SpinProps, 'size'> {
  center?: boolean;
  child?: React.FC<T>
  children?: React.ReactNode;
  conditionalRender?: boolean;
  inline?: boolean;
  loadableProps?: { [K in keyof T]: Loadable<T[K]> };
  size?: IconSize;
}

const Spinner = <T extends UnknownRecord>({
  center,
  className,
  conditionalRender,
  size,
  spinning,
  tip,
  loadableProps,
  child,
  ...props
}: Props<T>): ReactElement => {
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
        spinning={spinning}
        tip={tip}
        {...props}>
        {conditionalRender ? (spinning ? null : props.children) : props.children}
      </Spin>
    </div>
  );
};

export default Spinner;
