import React from 'react';

import Avatar, { Props as AvatarProps } from '../Avatar';

import css from './AvatarCard.module.scss';

export type Props = Omit<AvatarProps, 'hideTooltip'>;

const AvatarCard: React.FC<Props> = ({ className, displayName, ...avatarProps }: Props) => {
  return (
    <div className={`${css.base} ${className || ''}`}>
      <Avatar {...avatarProps} displayName={displayName} hideTooltip />
      <span>{displayName}</span>
    </div>
  );
};

export default AvatarCard;
