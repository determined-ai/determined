import React from 'react';

import Icon from 'shared/components/Icon';

import css from './GroupAvatar.module.scss';

type Props = {
  groupName: string | undefined;
};

const GroupAvatar: React.FC<Props> = ({ groupName }) => {
  return (
    <div className={css.group}>
      <Icon name="group" />
      <span>{groupName}</span>
    </div>
  );
};

export default GroupAvatar;
