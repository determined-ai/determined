import React, { useCallback, useEffect, useState } from 'react';

import Users from 'contexts/Users';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import { getUsers } from 'services/api';
import { EmptyParams } from 'services/types';
import { DetailedUser } from 'types';

const AppContexts: React.FC = () => {
  const [ canceler ] = useState(new AbortController());
  const setUsers = Users.useActionContext();
  const [ usersResponse, triggerUsersRequest ] =
    useRestApi<EmptyParams, DetailedUser[]>(getUsers, {});

  const fetchUsers = useCallback((): void => {
    triggerUsersRequest({ url: '/users' });
  }, [ triggerUsersRequest ]);

  usePolling(fetchUsers, { delay: 60000 });

  useEffect(() => {
    setUsers({ type: Users.ActionType.Set, value: usersResponse });
  }, [ usersResponse, setUsers ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return <React.Fragment />;
};

export default AppContexts;
