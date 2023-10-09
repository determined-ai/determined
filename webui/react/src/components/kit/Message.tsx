import React, { ReactNode } from 'react';

import Icon, { IconName } from 'components/kit/Icon';
import { XOR } from 'components/kit/internal/types';
import Header from 'components/kit/Typography/Header';

import { ValueOf } from './internal/types';
import css from './Message.module.scss';

export const MessageType = {
  Error: 'error',
  Info: 'info',
  Warning: 'warning',
} as const;

export type MessageType = ValueOf<typeof MessageType>;

export type Props = XOR<
  {
    description?: string;
    title: string;
    icon: ReactNode;
  }, {
    description?: string;
    title: string;
    type?: MessageType;
  }
>

const Message: React.FC<Props> = ({ description, title, icon, type }: Props) => {
  const getIcon = (type?: MessageType, icon?: ReactNode) => {
    if (type) {
      return <Icon decorative name={type as IconName} size="big" />;
    } else if (icon) {
      return icon;
    } else {
      return <Icon decorative name="info" size="big" />;
    }
  };

  return (
    <div className={css.base}>
      {getIcon(type, icon)}
      <Header>{title}</Header>
      {description && <span>{description}</span>}
    </div>
  );
};

export default Message;
