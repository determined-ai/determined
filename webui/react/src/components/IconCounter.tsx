import React from 'react';

import Icon from 'components/kit/Icon';
import { ValueOf } from 'shared/types';

import css from './IconCounter.module.scss';

const IconCounterType = {
  Active: 'active',
  Disabled: 'disabled',
} as const;

type IconCounterType = ValueOf<typeof IconCounterType>;

interface Props {
  count: number;
  name: string;
  onClick: () => void;
  type: IconCounterType;
}

const IconCounter: React.FC<Props> = (props: Props) => {
  const classes = [css.base];
  if (props.type) classes.push(css[props.type]);
  return (
    <a className={classes.join(' ')} onClick={props.onClick}>
      <Icon name={props.name} size="large" />
      <span className={css.count}>{props.count}</span>
    </a>
  );
};

export default IconCounter;
