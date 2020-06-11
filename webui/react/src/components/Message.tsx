import React from 'react';

import { CommonProps } from 'types';

import css from './Message.module.scss';

type Props = CommonProps

const Message: React.FC<Props> = ({ children }: Props) => {
  return (
    <div className={css.base}>
      {children}
    </div>
  );
};

export default Message;
