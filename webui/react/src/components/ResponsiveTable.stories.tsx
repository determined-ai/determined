import { Meta } from '@storybook/react';
import React from 'react';

import { generateAlphaNumeric, generateLetters } from 'shared/utils/string';

import ResponsiveTable from './ResponsiveTable';

export default {
  component: ResponsiveTable,
  parameters: {
    docs: { description: { component: 'Depricated. Prefer using InteractiveTable instead.' } },
    layout: 'padded',
  },
  title: 'Determined/Tables/ResponsiveTable',
} as Meta<typeof ResponsiveTable>;

const columns = new Array(20).fill(null).map(() => {
  const str = generateLetters();
  return {
    dataIndex: str,
    sorter: true,
    title: str,
  };
});

const data = new Array(100).fill(null).map(() => {
  const row: Record<string, string> = {};
  columns.forEach((column) => {
    row[column.dataIndex] = generateAlphaNumeric();
  });
  return row;
});

export const Default = (): React.ReactNode => (
  <ResponsiveTable columns={columns} dataSource={data} />
);
