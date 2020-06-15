import React from 'react';

import { CommonProps } from 'types';

import css from './Icon.module.scss';

interface Props extends CommonProps {
  name?: string;
  size?: 'tiny' | 'small' | 'medium' | 'large';
}

const defaultProps: Props = {
  name: 'star',
  size: 'medium',
};

const Icon: React.FC<Props> = ({ name, size, ...rest }: Props) => {
  const classes = [ css.base ];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);

  return <i className={classes.join(' ')} {...rest} />;
};

Icon.defaultProps = defaultProps;

export default Icon;
