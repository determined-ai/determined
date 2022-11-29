import { Tooltip } from 'antd';
import React, { CSSProperties } from 'react';

import { CommonProps } from '../../types';

import css from './Icon.module.scss';

export type IconSize =
  | 'tiny'
  | 'small'
  | 'medium'
  | 'large'
  | 'big'
  | 'great'
  | 'huge'
  | 'enormous'
  | 'giant'
  | 'jumbo'
  | 'mega';

export interface Props extends CommonProps {
  name?: string;
  size?: IconSize;
  style?: CSSProperties;
  title?: string;
}

const Icon: React.FC<Props> = ({
  name = 'star',
  size = 'medium',
  title,
  style,
  ...rest
}: Props) => {
  const classes = [css.base];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);

  const icon = <i className={classes.join(' ')} {...rest} style={style} />;
  return title ? <Tooltip title={title}>{icon}</Tooltip> : icon;
};

export default Icon;
