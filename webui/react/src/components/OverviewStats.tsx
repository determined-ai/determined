import React from 'react';

import css from './OverviewStats.module.scss';

interface Props {
  children: React.ReactNode;
  clickable?: boolean
  focused?: boolean;
  onClick?: () => void;
  title: string;
}

const OverviewStats: React.FC<Props> = (props: Props) => {
  const classes = [ css.base ];
  if (props.onClick || props.clickable) classes.push(css.clickable);
  if (props.focused) classes.push(css.focused);

  return (
    <div className={classes.join(' ')} onClick={props.onClick}>
      <div className={css.title}>{props.title}</div>
      <div className={css.info}>{props.children}</div>
    </div>
  );
};

export default OverviewStats;
