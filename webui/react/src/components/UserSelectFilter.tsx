import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import Auth from 'contexts/Auth';
import Users from 'contexts/Users';
import { User } from 'types';

import Icon from './Icon';
import css from './UserSelectFilter.module.scss';

const { Option } = Select;

interface Props {
  onChange: (value: SelectValue) => void;
  value?: SelectValue;
}

export const ALL_VALUE = 'all';

const userToSelectOption = (user: User): React.ReactNode =>
  <Option key={user.id} value={user.username}>{user.username}</Option>;

const UserSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();

  const handleFilter = useCallback((search: string, option) => {
    return option.props.children.indexOf(search) !== -1;
  }, []);

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
    <div className={css.base}>
      <div className={css.label}>Users</div>
      <Select
        defaultValue={value || ALL_VALUE}
        dropdownMatchSelectWidth={false}
        filterOption={handleFilter}
        optionFilterProp="children"
        showSearch={true}
        style={{ width: '10rem' }}
        suffixIcon={<Icon name="arrow-down" size="tiny" />}
        onSelect={handleSelect}>
        {options}
      </Select>
    </div>
  );
};

export default UserSelectFilter;
