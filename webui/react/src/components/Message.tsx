import { Empty } from 'antd';
import React from 'react';

import iconAlert from 'assets/icon-alert.svg';
import iconWarning from 'assets/icon-warning.svg';

import css from './Message.module.scss';

export enum MessageType {
  Alert = 'alert',
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
  type = MessageType.Alert,
}: Props) => {
  return (
    <div className={css.base}>
      {type === MessageType.Empty && Empty.PRESENTED_IMAGE_SIMPLE}
      {type === MessageType.Alert && <img src={iconAlert} />}
      {type === MessageType.Warning && <img src={iconWarning} />}
      <div className={css.title}>{title}</div>
      {message && <span>{message}</span>}
    </div>
  );
};

export default Message;
