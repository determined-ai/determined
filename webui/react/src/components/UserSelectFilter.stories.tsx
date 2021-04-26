import React, { useEffect, useState } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import StoreDecorator from 'storybook/StoreDecorator';

import UserSelectFilter from './UserSelectFilter';

export default {
  component: UserSelectFilter,
  decorators: [ StoreDecorator ],
  title: 'UserSelectFilter',
};

interface Props {
  value?: string;
}

const UserSelectFilterWithUsers: React.FC<Props> = ({ value }: Props) => {
  const [ currentValue, setCurrentValue ] = useState(value);
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({
      type: StoreAction.SetUsers,
      value: [
        { id: 1, isActive: true, isAdmin: true, username: 'admin' },
        { id: 2, isActive: true, isAdmin: false, username: 'user' },
        { id: 3, isActive: false, isAdmin: false, username: 'inactive' },
      ],
    });
  }, [ storeDispatch ]);

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
