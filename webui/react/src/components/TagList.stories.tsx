import { Meta } from '@storybook/react';
import React, { useState } from 'react';

import EditableTagList from './TagList';

export default {
  component: EditableTagList,
  title: 'Determined/EditableTagList',
} as Meta<typeof EditableTagList>;

const DEFAULT_TAGS = ['hello', 'world'];

export const Default = (): React.ReactElement => {
  const [tags, setTags] = useState(DEFAULT_TAGS);

  return <EditableTagList tags={tags} onChange={setTags} />;
};
