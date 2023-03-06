import React, { ReactNode } from 'react';

import css from './Nameplate.module.scss';

export interface Props {
  alias?: string;
  className?: string;
  compact?: boolean;
  icon: ReactNode;
  name?: string;
}

const Nameplate: React.FC<Props> = ({ alias, className, compact, icon, name }) => {
  const classnames = [css.avatarCard];
  if (compact) classnames.push(css.compact);
  if (className) classnames.push(className);

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
