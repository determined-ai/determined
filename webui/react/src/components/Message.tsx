import { Empty } from 'antd';
import React from 'react';

import warningImage from 'assets/warning.svg';

import css from './Message.module.scss';

export enum MessageType {
  Empty = 'empty',
  Warning = 'warning',
}

interface Props {
  message?: string;
  title: string;
  type?: MessageType;
}

const Message: React.FC<Props> = ({
  message,
  title,
  type = MessageType.Warning,
}: Props) => {
  return (
    <div className={css.base}>
      {type === MessageType.Empty && Empty.PRESENTED_IMAGE_SIMPLE}
      {type === MessageType.Warning && <img src={warningImage} />}
      <div className={css.title}>{title}</div>
      {message && <span>{message}</span>}
    </div>
  );
};

export default Message;
