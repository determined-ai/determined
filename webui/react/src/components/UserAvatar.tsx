import React from 'react';

import Avatar, { Props as AvatarProps } from 'components/kit/Avatar';
import { User } from 'types';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  user?: User;
}

const UserAvatar: React.FC<Props> = ({ user, ...rest }) => {
  const displayName = getDisplayName(user);

  return <Avatar {...rest} displayName={displayName} />;
};

export default UserAvatar;
