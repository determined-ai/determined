import React, { PropsWithChildren } from 'react';

import css from './OverviewStats.module.scss';

interface Props {
  title: string;
}

const OverviewStats: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return (
    <div className={css.base}>
      <div className={css.title}>{props.title}</div>
      <div className={css.info}>{props.children}</div>
    </div>
  );
};

export default OverviewStats;
