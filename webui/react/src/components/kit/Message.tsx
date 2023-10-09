import React, { ReactNode } from 'react';

import Icon, { IconName } from 'components/kit/Icon';
import { XOR } from 'components/kit/internal/types';

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
    body?: string;
    title: string;
    icon: ReactNode;
  }, {
    body?: string;
    title: string;
    type: MessageType;
  }
>

const Message: React.FC<Props> = ({ body, title, icon, type }: Props) => {
  const getIcon = (type?: MessageType, icon?: ReactNode) => {
    if (type) {
      return <Icon decorative name={type as IconName} size="big" />;
    } else {
      return icon;
    }
  };

  return (
    <div className={css.base}>
      {getIcon(type, icon)}
      <div className={css.title}>{title}</div>
      {body && <span>{body}</span>}
    </div>
  );
};

export default Message;
