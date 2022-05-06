import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { useStore } from 'contexts/Store';
import { ALL_VALUE, User } from 'types';
import { getDisplayName } from 'utils/user';

import SelectFilter from './SelectFilter';

const { Option } = Select;

interface Props {
  onChange?: (value: SelectValue) => void;
  value?: SelectValue;
}

const userToSelectOption = (user: User): React.ReactNode =>
  <Option key={user.id} value={user.id}>{getDisplayName(user)}</Option>;

const UserSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const { auth, users } = useStore();

  const handleSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;
    const singleValue = Array.isArray(newValue) ? newValue[0] : newValue;
    onChange(singleValue);
  }, [ onChange ]);

  const options = useMemo(() => {
    const authUser = auth.user;
    const list: React.ReactNode[] = [ <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option> ];

    if (authUser) {
      list.push(
        <Option key={authUser.id} value={authUser.id}>
          {getDisplayName(authUser)}
        </Option>,
      );
    }

    if (users) {
      const allOtherUsers = users
        .filter(user => (!authUser || user.id !== authUser.id))
        .sort((a, b) => getDisplayName(a).localeCompare(getDisplayName(b), 'en'))
        .map(userToSelectOption);
      list.push(...allOtherUsers);
    }

    return list;
  }, [ auth.user, users ]);

  return (
    <SelectFilter
      dropdownMatchSelectWidth={200}
      label="Users"
      style={{ maxWidth: 200 }}
      value={value || ALL_VALUE}
      onSelect={handleSelect}>
      {options}
    </SelectFilter>
  );
};

export default UserSelectFilter;
