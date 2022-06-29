import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import Avatar, { Props } from 'shared/components/Avatar';
import { getDisplayName } from 'utils/user';

const UserAvatar: React.FC<{
  name?: string;
  userId?: number;
} & Omit<Props, 'displayName'>> = ({ name, userId, ...rest }) => {

  const [ displayName, setDisplayName ] = useState('');
  const { users } = useStore();
  const fetchUsers = useFetchUsers(new AbortController());

  useEffect(() => {
    if (!name && userId) {
      if (!users.length) {
        fetchUsers();
      }
      const user = users.find(user => user.id === userId);
      setDisplayName(getDisplayName(user));
    } else if (name) {
      setDisplayName(name);
    }
  }, [ fetchUsers, userId, name, users ]);

  return <Avatar {...rest} displayName={name || displayName} />;
};

export default UserAvatar;
