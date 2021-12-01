import React from 'react';

import EditableMetadata from './EditableMetadata';

export default {
  component: EditableMetadata,
  parameters: { layout: 'centered' },
  title: 'EditableMetadata',
};

const metadata = { key: 'value', lorem: 'ipsum', test: 'component' };

export const Default = (): React.ReactNode => (
  <EditableMetadata editing={false} metadata={metadata} />
);

export const Editing = (): React.ReactNode => (
  <EditableMetadata editing={true} metadata={metadata} />
);
