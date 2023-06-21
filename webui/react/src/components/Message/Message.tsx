import React from 'react';

import * as Images from 'components/Image';
import useUI from 'stores/contexts/UI';
import { ValueOf } from 'types';

import css from './Message.module.scss';

export const MessageType = {
  Alert: 'alert',
  Empty: 'empty',
  Warning: 'warning',
} as const;

export type MessageType = ValueOf<typeof MessageType>;

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

const Message: React.FC<Props> = ({ message, style, title, type = MessageType.Alert }: Props) => {
  const { ui } = useUI();
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
