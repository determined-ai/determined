import React from 'react';

import { DarkLight } from 'shared/themes';

import AvatarCard from './AvatarCard';

export default {
  component: AvatarCard,
  title: 'Avatar Card',
};

export const Default = (): React.ReactNode => (
  <AvatarCard darkLight={DarkLight.Light} displayName="Admin" />
);

export const DarkMode = (): React.ReactNode => (
  <AvatarCard darkLight={DarkLight.Dark} displayName="Admin" />
);

export const TwoWordName = (): React.ReactNode => (
  <AvatarCard darkLight={DarkLight.Light} displayName="Determined AI" />
);

export const ThreeWordName = (): React.ReactNode => (
  <AvatarCard darkLight={DarkLight.Light} displayName="Gold Experience Requiem" />
);
