import { ComponentStory } from '@storybook/react';
import React from 'react';

import EditableMetadata from './EditableMetadata';

export default {
  component: EditableMetadata,
  title: 'Determined/EditableMetadata',
};

const metadata = { key: 'value', lorem: 'ipsum', test: 'component' };

export const Default: ComponentStory<typeof EditableMetadata> = (args) => (
  <EditableMetadata {...args} metadata={metadata} />
);

export const Editing = (): React.ReactNode => (
  <EditableMetadata editing={true} metadata={metadata} />
);

Default.args = { editing: false };
