import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import SelectFilter from 'components/kit/SelectFilter';
import { useCurrentUser, useUsers } from 'stores/users';
import { ALL_VALUE, User } from 'types';
import { Loadable } from 'utils/loadable';
import { getDisplayName } from 'utils/user';

const { Option } = Select;

interface Props {
  onChange?: (value: SelectValue) => void;
  value?: SelectValue;
}

const userToSelectOption = (user: User): React.ReactNode => (
  <Option key={user.id} value={user.id}>
    {getDisplayName(user)}
  </Option>
);

const UserSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const users = Loadable.match(useUsers(), {
    Loaded: (cUser) => cUser.users,
    NotLoaded: () => [],
  }); // TODO: handle loading state // TODO: handle loading state
  const loadableCurrentUser = useCurrentUser();
  const authUser = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });

  const handleSelect = useCallback(
    (newValue: SelectValue) => {
      if (!onChange) return;
      const singleValue = Array.isArray(newValue) ? newValue[0] : newValue;
      onChange(singleValue);
    },
    [onChange],
  );

  const options = useMemo(() => {
    const list: React.ReactNode[] = [
      <Option key={ALL_VALUE} value={ALL_VALUE}>
        All
      </Option>,
    ];

    if (authUser) {
      list.push(
        <Option key={authUser.id} value={authUser.id}>
          {getDisplayName(authUser)}
        </Option>,
      );
    }

    if (users) {
      const allOtherUsers = users
        .filter((user) => !authUser || user.id !== authUser.id)
        .sort((a, b) => getDisplayName(a).localeCompare(getDisplayName(b), 'en'))
        .map(userToSelectOption);
      list.push(...allOtherUsers);
    }

    return list;
  }, [authUser, users]);

  return (
    <SelectFilter
      label="Users"
      value={value || ALL_VALUE}
      onSelect={handleSelect}>
      {options}
    </SelectFilter>
  );
};

export default UserSelectFilter;
