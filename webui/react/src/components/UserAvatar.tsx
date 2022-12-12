import React from 'react';

import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import useUI from 'shared/contexts/stores/UI';
import { useUsers } from 'stores/users';
import { Loadable } from 'utils/loadable';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  userId?: number;
}

const UserAvatar: React.FC<Props> = ({ userId, ...rest }) => {
  const users = Loadable.getOrElse([], useUsers()); // TODO: handle loading state
  const { ui } = useUI();
  const displayName = getDisplayName(users.find((user) => user.id === userId));

  return <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
};

export default UserAvatar;
