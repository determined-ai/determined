import { ComponentStory, Meta } from '@storybook/react';
import React, { useCallback, useState } from 'react';

import loremIpsum from 'shared/utils/loremIpsum';

import InlineEditor from './InlineEditor';

export default {
  argTypes: { allowNewline: { description: 'allow newline (use [shift] + [enter])' } },
  component: InlineEditor,
  title: 'Determined/InlineEditor',
} as Meta<typeof InlineEditor>;

export const Default: ComponentStory<typeof InlineEditor> = (args) => {
  const [value, setValue] = useState('Edit Me!');

  const save = useCallback((newValue: string): Promise<void> => {
    return new Promise<void>((resolve) => {
      setTimeout(() => {
        setValue(newValue);
        resolve();
      }, 1500);
    });
  }, []);

  const handleSave = useCallback(
    async (newValue: string) => {
      await save(newValue);
    },
    [save],
  );

  return <InlineEditor {...args} value={value} onSave={handleSave} />;
};

export const LargeText: ComponentStory<typeof InlineEditor> = (args) => (
  <InlineEditor {...args} value={loremIpsum} />
);

Default.args = { allowNewline: false, maxLength: 100, placeholder: 'placeholder text' };
