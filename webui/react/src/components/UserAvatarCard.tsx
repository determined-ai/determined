import React from 'react';

import AvatarCard from 'shared/components/AvatarCard/AvatarCard';
import { DetailedUser, User } from 'types';
import { getDisplayName } from 'utils/user';

export interface Props {
  className?: string;
  user?: DetailedUser | User
}

const UserAvatarCard: React.FC<Props> = ({ className, user }) => {
  return <AvatarCard className={className} displayName={getDisplayName(user)} />;
};

export default UserAvatarCard;
