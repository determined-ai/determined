import React from 'react';

import AvatarCard from 'shared/components/AvatarCard/AvatarCard';
import { DarkLight } from 'shared/themes';
import { DetailedUser, User } from 'types';
import { getDisplayName } from 'utils/user';

export interface Props {
  className?: string;
  darkLight: DarkLight;
  user?: DetailedUser | User;
}

const UserAvatarCard: React.FC<Props> = ({ className, darkLight, user }) => (
  <AvatarCard className={className} darkLight={darkLight} displayName={getDisplayName(user)} />
);

export default UserAvatarCard;
