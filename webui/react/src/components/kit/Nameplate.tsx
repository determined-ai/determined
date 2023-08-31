import React, { ReactNode } from 'react';

import css from './Nameplate.module.scss';

export interface Props {
  alias?: string;
  compact?: boolean;
  icon: ReactNode;
  name: string;
}

const Nameplate: React.FC<Props> = ({ alias, compact, icon, name }) => {
  const classnames = [css.base];
  if (compact) classnames.push(css.compact);

  return (
    <div className={classnames.join(' ')}>
      <div>
        {/* icon needs wrapper to maintain width */}
        {icon}
      </div>
      <div className={css.text}>
        {alias && <div className={css.alias}>{alias}</div>}
        <div className={css.name}>{name}</div>
      </div>
    </div>
  );
};

export default Nameplate;
