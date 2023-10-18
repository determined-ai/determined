import React, { ReactNode } from 'react';

import Icon, { IconName } from 'components/kit/Icon';
import Header from 'components/kit/Typography/Header';

import css from './Message.module.scss';

interface base {
  action?: React.ReactElement;
  icon?: IconName | React.ReactElement;
}
interface descriptionRequired extends base {
  description: ReactNode;
  title?: string;
}
interface titleRequired extends base {
  title: string;
  description?: ReactNode;
}

export type Props = descriptionRequired | titleRequired;

const Message: React.FC<Props> = ({ action, description, title, icon }: Props) => {
  const getIcon = (icon?: IconName | React.ReactElement) => {
    if (typeof icon === 'string') {
      return <Icon decorative name={icon as IconName} size="jumbo" />;
    } else {
      return icon;
    }
  };

  return (
    <div className={css.base}>
      {icon && getIcon(icon)}
      {title && <Header>{title}</Header>}
      {description && <p className={css.description}>{description}</p>}
      {action}
    </div>
  );
};

export default Message;
