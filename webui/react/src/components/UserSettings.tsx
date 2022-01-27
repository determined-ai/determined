import React from 'react';

import Avatar from 'components/Avatar';

import css from './UserSettings.module.scss';

interface Props {
  username: string;
}

const UserSettings: React.FC<Props> = ({ username }: Props) => {
  return (
    <div className={css.base}>
      <div className={css.field}>
        <span className={css.label}>Avatar</span>
        <span className={css.value}>
          <Avatar hideTooltip large name={username} />
        </span>
      </div>
      <div className={css.field}>
        <span className={css.label}>Username</span>
        <span className={css.value}>{username}</span>
      </div>
    </div>
  );
};

export default UserSettings;
