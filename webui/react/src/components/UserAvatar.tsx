import React from 'react';

import { useStore } from 'contexts/Store';
import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  userId?: number;
}

const UserAvatar: React.FC<Props> = ({ userId, ...rest }) => {
  const { ui, users } = useStore();
  const displayName = getDisplayName(users.find((user) => user.id === userId));

  return <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
};

export default UserAvatar;
