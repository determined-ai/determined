import React, { ReactNode } from 'react';

import Icon from 'shared/components/Icon';

import css from './Empty.module.scss';

interface EmptyProps {
  description?: ReactNode;
  icon?: string;
  title?: string;
}

const Empty: React.FC<EmptyProps> = ({ icon, title, description }: EmptyProps) => {
  return (
    <div className={css.emptyBase}>
      {icon ? (
        <div className={css.icon}>
          <Icon name={icon} size="mega" />
        </div>
      ) : null}
      <h4>{title}</h4>
      <p className={css.description}>{description}</p>
    </div>
  );
};

export default Empty;
