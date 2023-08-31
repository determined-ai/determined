import React, { ReactNode } from 'react';

import Icon, { IconName } from 'components/kit/Icon';

import css from './Empty.module.scss';

interface EmptyProps {
  description?: ReactNode;
  icon?: IconName;
  title?: string;
}

const Empty: React.FC<EmptyProps> = ({ icon, title, description }: EmptyProps) => {
  return (
    <div className={css.emptyBase}>
      {icon ? (
        <div className={css.icon}>
          <Icon decorative name={icon} size="mega" />
        </div>
      ) : null}
      {title ? <h4>{title}</h4> : null}
      {description ? <p className={css.description}>{description}</p> : null}
    </div>
  );
};

export default Empty;
