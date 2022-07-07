import React from 'react';

import { DarkLight } from 'shared/themes';

import Avatar from '../Avatar';

import css from './AvatarCard.module.scss';

export interface Props {
  className?: string;
  darkLight: DarkLight;
  displayName: string;
}

const AvatarCard: React.FC<Props> = ({ className, darkLight, displayName }: Props) => {
  return (
    <div className={`${css.base} ${className || ''}`}>
      <Avatar darkLight={darkLight} displayName={displayName} hideTooltip />
      <span>{displayName}</span>
    </div>
  );
};

export default AvatarCard;
