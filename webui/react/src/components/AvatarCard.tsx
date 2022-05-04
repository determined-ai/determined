import React from 'react';

import { DetailedUser, User } from 'types';
import { getDisplayName } from 'utils/user';

import Avatar from './Avatar';
import css from './AvatarCard.module.scss';

interface Props {
  className?: string;
  user?: DetailedUser | User
}

const AvatarCard: React.FC<Props> = ({ className, user }: Props) => {
  return (
    <div className={`${css.base} ${className || ''}`}>
      <Avatar hideTooltip userId={user?.id} />
      <span>{getDisplayName(user)}</span>
    </div>
  );
};

export default AvatarCard;
