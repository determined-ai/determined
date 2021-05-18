import React, { PropsWithChildren } from 'react';

import css from './OverviewStats.module.scss';

interface Props {
  onClick?: () => void,
  title: string;
}

const OverviewStats: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  if (props.onClick) classes.push(css.baseClickable);

  return (
    <div className={classes.join(' ')} onClick={props.onClick}>
      <div className={css.title}>{props.title}</div>
      <div className={css.info}>{props.children}</div>
    </div>
  );
};

export default OverviewStats;
