import React from 'react';

import Icon from 'components/Icon';

import css from './IconCounter.module.scss';

interface Props {
  name: string;
  count: number;
  type: IconCounterType;
  onClick: () => void;
}

enum IconCounterType {
  Active = 'active',
  Disabled = 'disabled',
}

const IconCounter: React.FC<Props> = (props: Props) => {
  const classes = [ css.base ];
  if (props.type) classes.push(css[props.type]);
  return (
    <a className={classes.join(' ')} onClick={props.onClick}>
      <Icon name={props.name} size="large" />
      <span className={css.count}>{props.count}</span>
    </a>
  );
};

export default IconCounter;
