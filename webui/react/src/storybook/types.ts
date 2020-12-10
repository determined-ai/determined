import { addDecorator } from '@storybook/react';

export type DecoratorFunction = Parameters<typeof addDecorator>[0];

export interface StoryMetadata {
  component: React.ReactNode;
  decorators?: DecoratorFunction[];
  title: string;
}
