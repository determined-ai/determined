import { boolean, number, text, withKnobs } from '@storybook/addon-knobs';
import React, { useCallback, useState } from 'react';

import loremIpsum from 'storybook/loremIpsum';

import InlineEditor from './InlineEditor';

export default {
  component: InlineEditor,
  decorators: [ withKnobs ],
  title: 'InlineEditor',
};

export const Default = (): React.ReactNode => {
  const [ value, setValue ] = useState('Edit Me!');

  const save = useCallback((newValue: string): Promise<void> => {
    return new Promise<void>(resolve => {
      setTimeout(() => {
        setValue(newValue);
        resolve();
      }, 1500);
    });
  }, []);

  const handleSave = useCallback(async (newValue: string) => {
    await save(newValue);
  }, [ save ]);

  return (
    <InlineEditor
      allowNewline={boolean('allow newline (use <shift> + <enter>)', false)}
      isOnDark={boolean('is on dark', false)}
      maxLength={number('max length', 100)}
      placeholder={text('placeholder', 'placeholder text')}
      value={value}
      onSave={handleSave}
    />
  );
};

export const LargeText = (): React.ReactNode => (
  <InlineEditor value={loremIpsum} />
);

export const IsOnDark = (): React.ReactNode => (
  <InlineEditor isOnDark value="Hello Darkness" />
);
IsOnDark.parameters = {
  backgrounds: {
    default: 'dark background',
    values: [
      { name: 'dark background', value: '#111' },
    ],
  },
};
