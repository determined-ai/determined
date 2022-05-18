import { Tooltip } from 'antd';
import React from 'react';

import { CommonProps } from '../../types';

import css from './Icon.module.scss';

export type IconSize = (
  'tiny' |
  'small' |
  'medium' |
  'large' |
  'big' |
  'great' |
  'huge' |
  'enormous' |
  'giant' |
  'jumbo' |
  'mega'
);

interface Props extends CommonProps {
  name?: string;
  size?: IconSize;
  title?: string;
}

const Icon: React.FC<Props> = ({ name = 'star', size = 'medium', title, ...rest }: Props) => {
  const classes = [ css.base ];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);

  const icon = <i className={classes.join(' ')} {...rest} />;

  return title ? <Tooltip title={title}>{icon}</Tooltip> : icon;
};

export default Icon;
