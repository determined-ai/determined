import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import Auth from 'contexts/Auth';
import Users from 'contexts/Users';
import { ALL_VALUE, User } from 'types';

import SelectFilter from './SelectFilter';

const { Option } = Select;

interface Props {
  onChange: (value: SelectValue) => void;
  value?: SelectValue;
}

const userToSelectOption = (user: User): React.ReactNode =>
  <Option key={user.id} value={user.username}>{user.username}</Option>;

const UserSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();

  const handleSelect = useCallback((newValue: SelectValue) => {
    const singleValue = Array.isArray(newValue) ? newValue[0] : newValue;
    onChange(singleValue);
  }, [ onChange ]);

  const options = useMemo(() => {
    const authUser = auth.user;
    const list: React.ReactNode[] = [ <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option> ];

    if (authUser) {
      list.push(<Option key={authUser.id} value={authUser.username}>{authUser.username}</Option>);
    }

    if (users.data) {
      const allOtherUsers = users.data
        .filter(user => (!authUser || user.id !== authUser.id))
        .sort((a, b) => a.username.localeCompare(b.username, 'en'))
        .map(userToSelectOption);
      list.push(...allOtherUsers);
    }

    return list;
  }, [ auth.user, users.data ]);

  return (
    <SelectFilter label="Users" value={value || ALL_VALUE} onSelect={handleSelect}>
      {options}
    </SelectFilter>
  );
};

export default UserSelectFilter;
