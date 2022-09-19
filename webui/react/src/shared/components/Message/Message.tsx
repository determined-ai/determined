import React from 'react';

import { useStore } from 'contexts/Store';
import * as Images from 'shared/components/Image';

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

const IMAGE_MAP = {
  [MessageType.Alert]: Images.ImageAlert,
  [MessageType.Empty]: Images.ImageEmpty,
  [MessageType.Warning]: Images.ImageWarning,
};

const Message: React.FC<Props> = ({
  message,
  style,
  title,
  type = MessageType.Alert,
}: Props) => {
  const { ui } = useStore();
  const ImageComponent = IMAGE_MAP[type];
  return (
    <div className={css.base} style={style}>
      <ImageComponent darkLight={ui.darkLight} />
      <div className={css.title}>{title}</div>
      {message && <span>{message}</span>}
    </div>
  );
};

export default Message;
