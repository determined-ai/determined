import { MinusOutlined, PlusOutlined } from '@ant-design/icons';
import { Select as AntdSelect } from 'antd';
import React, { useMemo, useState } from 'react';

import { User } from 'components/kit/internal/types';

import Button from './Button';
import css from './Facepile.module.scss';
import UserAvatar from './UserAvatar';

export interface Props {
  editable?: boolean;
  onAddUser?: (user: User) => void;
  selectableUsers?: User[]; // This prop should be used to pass as options to the dropdown.
  users?: User[];
}

const Facepile: React.FC<Props> = ({
  editable = false,
  selectableUsers = [],
  users = [],
  onAddUser,
}) => {
  const [showDropdown, setShowDropdown] = useState(false);
  const [showAllAvatars, setShowAllAvatars] = useState(false);
  const [avatars, setAvatars] = useState(users);
  const amountOfAvatars = useMemo(() => avatars.length, [avatars.length]);
  const usersItems = useMemo(
    () =>
      selectableUsers
        .filter((user) => !avatars.find((av) => av.id === user.id))
        .map((user) => ({
          label: user.username,
          value: user.id,
        })),
    [selectableUsers, avatars],
  );
  const visibleAvatars = useMemo(
    () => (showAllAvatars ? avatars : avatars.slice(0, 5)),
    [avatars, showAllAvatars],
  );
  const showButton = useMemo(() => editable || amountOfAvatars > 5, [editable, amountOfAvatars]);
  const buttonLabel = useMemo(() => {
    if (amountOfAvatars > 5 && !showAllAvatars) return `+ ${amountOfAvatars - 5}`;
  }, [amountOfAvatars, showAllAvatars]);
  const buttonIcon = useMemo(() => {
    if (showAllAvatars || showDropdown) return <MinusOutlined />;
    if (editable) return <PlusOutlined />;
  }, [showAllAvatars, showDropdown, editable]);

  return (
    <div className={css.container}>
      {visibleAvatars.map((avatar) => (
        <UserAvatar className={css.spacing} key={avatar.id} user={avatar} />
      ))}
      {showDropdown && (
        <AntdSelect
          filterOption={(input, option) =>
            (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
          }
          options={usersItems}
          placeholder="Select a user"
          placement="bottomRight"
          onChange={(value) => {
            // we know that it will find a user since the options are based on that variable
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const user = selectableUsers.find((u) => u.id === value)!;

            setAvatars((prev) => {
              const newAvatars = [...prev];

              newAvatars.push(user);

              return newAvatars;
            });
            setShowDropdown(false);
            onAddUser?.(user);
          }}
        />
      )}
      {showButton && (
        <span className={css.addButton}>
          <Button
            icon={buttonIcon}
            type="text"
            onClick={() => {
              if (amountOfAvatars > 5) {
                if (showAllAvatars && editable) {
                  setShowDropdown(true);
                  return;
                }

                setShowAllAvatars((prev) => !prev);
                return;
              }

              setShowDropdown((prev) => !prev);
            }}>
            {buttonLabel}
          </Button>
          {showAllAvatars && editable && (
            <Button
              icon={<MinusOutlined />}
              type="text"
              onClick={() => {
                setShowAllAvatars(false);
                setShowDropdown(false);
              }}>
              Hide
            </Button>
          )}
        </span>
      )}
    </div>
  );
};

export default Facepile;
