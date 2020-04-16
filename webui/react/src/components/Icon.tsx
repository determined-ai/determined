import React from 'react';

import css from './Icon.module.scss';

interface Props {
  name?: string;
  size?: 'tiny' | 'small' | 'medium' | 'large';
}

const defaultProps: Props = {
  name: 'star',
  size: 'medium',
};

const Icon: React.FC<Props> = ({ name, size }: Props) => {
  const classes = [ css.base ];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);

  return <i className={classes.join(' ')} />;
};

Icon.defaultProps = defaultProps;

export default Icon;
