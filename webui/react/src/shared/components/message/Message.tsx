import { Empty } from 'antd';
import React from 'react';

import iconAlert from 'shared/assets/images/icon-alert.svg';
import iconWarning from 'shared/assets/images/icon-warning.svg';

import css from './Message.module.scss';

export enum MessageType {
  Alert = 'alert',
  Empty = 'empty',
  Warning = 'warning',
}

export interface Props {
  message?: string;
  style?: React.CSSProperties;
  title: string;
  type?: MessageType;
}

const Message: React.FC<Props> = ({
  message,
  style,
  title,
  type = MessageType.Alert,
}: Props) => {
  return (
    <div className={css.base} style={style}>
      {type === MessageType.Empty && Empty.PRESENTED_IMAGE_SIMPLE}
      {type === MessageType.Alert && <img alt={MessageType.Alert} src={iconAlert} />}
      {type === MessageType.Warning && <img alt={MessageType.Warning} src={iconWarning} />}
      <div className={css.title}>{title}</div>
      {message && <span>{message}</span>}
    </div>
  );
};

export default Message;
