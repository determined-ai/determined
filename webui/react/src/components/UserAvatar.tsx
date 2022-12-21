import React from 'react';

import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import useUI from 'shared/contexts/stores/UI';
import { DetailedUser } from 'types';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  user?: DetailedUser;
}

const UserAvatar: React.FC<Props> = ({ user, ...rest }) => {
  const { ui } = useUI();
  const displayName = getDisplayName(user);

  return <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
};

export default UserAvatar;
