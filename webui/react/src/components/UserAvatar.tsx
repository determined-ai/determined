import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  userId?: number;
}

const UserAvatar: React.FC<Props> = ({ userId, ...rest }) => {
  const [ displayName, setDisplayName ] = useState('');
  const { ui, users } = useStore();
  const fetchUsers = useFetchUsers(new AbortController());

  useEffect(() => {
    if (!userId) return;
    if (!users.length) fetchUsers();
    const user = users.find(user => user.id === userId);
    setDisplayName(getDisplayName(user));
  }, [ fetchUsers, userId, users ]);

  return <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
};

export default UserAvatar;
