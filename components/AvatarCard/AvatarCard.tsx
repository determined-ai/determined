import React from 'react';

import Avatar from '../Avatar';

import css from './AvatarCard.module.scss';

export interface Props {
  className?: string;
  displayName: string;
}

const AvatarCard: React.FC<Props> = ({ className, displayName }: Props) => {
  return (
    <div className={`${css.base} ${className || ''}`}>
      <Avatar displayName={displayName} hideTooltip />
      <span>{displayName}</span>
    </div>
  );
};

export default AvatarCard;
