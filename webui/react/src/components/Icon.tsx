import { Tooltip } from 'antd';
import React from 'react';

import { CommonProps } from 'types';

import css from './Icon.module.scss';

export type IconSize = 'tiny' | 'small' | 'medium' | 'large' | 'big';

interface Props extends CommonProps {
  name?: string;
  size?: IconSize;
  title?: string;
}

const defaultProps: Props = {
  name: 'star',
  size: 'medium',
};

const Icon: React.FC<Props> = ({ name, title, size, ...rest }: Props) => {
  const classes = [ css.base ];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);
  const icon = <i className={classes.join(' ')} {...rest} />;

  if (title) {
    return <Tooltip title={title}>
      {icon}
    </Tooltip>;
  }
  return icon;
};

Icon.defaultProps = defaultProps;

export default Icon;
