import React from 'react';

import EditableMetadata from './EditableMetadata';

export default {
  component: EditableMetadata,
  decorators: [
    (Story: React.FC): React.ReactNode => (
      <div style={{ border: '1px solid black', padding: '20px' }}>
        <Story />
      </div>
    ),
  ],
  parameters: { layout: 'centered' },
  title: 'Editable Metadata',
};

const metadata = { key: 'value', lorem: 'ipsum', test: 'component' };

export const Default = (): React.ReactNode => (
  <EditableMetadata editing={false} metadata={metadata} />
);

export const Editing = (): React.ReactNode => (
  <EditableMetadata editing={true} metadata={metadata} />
);
