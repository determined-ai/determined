import React, { useMemo, useState } from 'react';

import SelectFilter from 'components/SelectFilter';
import { useUsers } from 'stores/users';
import { DetailedUser } from 'types';
import { Loadable } from 'utils/loadable';

import Button from './Button';
import css from './Facepile.module.scss';
import UserAvatar from './UserAvatar';

export interface Props {
  editable?: boolean;
  users?: DetailedUser[];
  // TODO: add vertical orientation
}

const Facepile: React.FC<Props> = ({ editable = false, users = [] }) => {
  const [showDropdown, setShowDropdown] = useState(false);
  const [showAllAvatars, setShowAllAvatars] = useState(false);
  const [avatars, setAvatars] = useState(users);
  const loadableUsers = useUsers();
  const loadedUsers = Loadable.match(loadableUsers, {
    Loaded: (u) => u.users,
    NotLoaded: () => [],
  });
  const amountOfAvatars = useMemo(() => avatars.length, [avatars.length]);
  const usersItems = useMemo(
    () =>
      loadedUsers.map((user) => ({
        label: user.username,
        value: user.id,
      })),
    [loadedUsers],
  );
  const visibleAvatars = useMemo(() => showAllAvatars ? avatars : avatars.slice(0, 5), [avatars, showAllAvatars]);
  const showButton = useMemo(() => editable || amountOfAvatars > 5, [editable, amountOfAvatars]);
  const buttonLabel = useMemo(() => {
    if (amountOfAvatars > 5 && !showAllAvatars) return `+ ${amountOfAvatars - 5}`;

    return editable ? 'Add' : 'Hide';
  }, [amountOfAvatars, showAllAvatars, editable]);

  return (
    <div className={css.container}>
      {
        visibleAvatars.map((avatar) => <UserAvatar className={css.spacing} key={avatar.id} user={avatar} />)
      }
      {showDropdown && (
        <SelectFilter
          className={css.spacing}
          filterOption={(input, option) =>
            (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
          }
          options={usersItems}
          placeholder="Select a user"
          placement="bottomRight"
          showSearch={true}
          onChange={(value) => {
            // we know that it will find a user since the options are based on that variable
            // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
            const user = loadedUsers.find((u) => u.id === value)!;

            setAvatars((prev) => {
              const newAvatars = [...prev];

              newAvatars.push(user);

              return newAvatars;
            });
            setShowDropdown(false);
          }}
        />
      )}
      {showButton && (
        <span className={css.addButton}>
          <Button
            type={!buttonLabel.includes('+') ? 'primary' : 'text'}
            onClick={() => {
              if (amountOfAvatars > 5) {
                if (showAllAvatars && editable) {
                  setShowDropdown(true);
                  return;
                };

                setShowAllAvatars((prev) => !prev);
                return;
              }

              setShowDropdown((prev) => !prev);
            }}>
            {buttonLabel}
          </Button>
          {(showAllAvatars && editable) && (
            <Button
              type="primary"
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
