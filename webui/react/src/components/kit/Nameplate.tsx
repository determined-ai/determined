import React, { ReactNode } from 'react';

import css from './Nameplate.module.scss';

export interface Props {
  alias?: string;
  compact?: boolean;
  icon: ReactNode;
  name: string;
}

const Nameplate: React.FC<Props> = ({ alias, compact, icon, name }) => {
  const classnames = [css.avatarCard];
  if (compact) classnames.push(css.compact);

  return (
    <div className={classnames.join(' ')}>
      {icon}
      <div className={css.text}>
        {alias && <span className={css.alias}>{alias}</span>}
        {<span>{name}</span>}
      </div>
    </div>
  );
};

export default Nameplate;
