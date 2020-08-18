import React from 'react';

import warningImage from 'assets/warning.svg';

import css from './Message.module.scss';

interface Props {
  message?: string;
  title: string;
}

const Message: React.FC<Props> = ({ message, title }: Props) => {
  return (
    <div className={css.base}>
      <img src={warningImage} />
      <div className={css.title}>{title}</div>
      {message && <span>{message}</span>}
    </div>
  );
};

export default Message;
