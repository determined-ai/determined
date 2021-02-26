import React, { useEffect, useState } from 'react';

import Users from 'contexts/Users';
import { AuthDecorator, UsersDecorator } from 'storybook/ContextDecorators';

import UserSelectFilter from './UserSelectFilter';

export default {
  component: UserSelectFilter,
  decorators: [ AuthDecorator, UsersDecorator ],
  title: 'UserSelectFilter',
};

interface Props {
  value?: string;
}

const UserSelectFilterWithUsers: React.FC<Props> = ({ value }: Props) => {
  const [ currentValue, setCurrentValue ] = useState(value);
  const setUsers = Users.useActionContext();

  useEffect(() => {
    setUsers({
      type: Users.ActionType.Set,
      value: [
        { isActive: true, isAdmin: true, username: 'admin' },
        { isActive: true, isAdmin: false, username: 'user' },
        { isActive: false, isAdmin: false, username: 'inactive' },
      ],
    });
  }, [ setUsers ]);

  return (
    <UserSelectFilter
      value={currentValue}
      onChange={(newValue) => setCurrentValue(newValue as string)}
    />
  );
};

export const Default = (): React.ReactNode => (
  <UserSelectFilterWithUsers />
);
