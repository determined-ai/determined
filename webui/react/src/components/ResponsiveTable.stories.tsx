import React from 'react';

import { generateAlphaNumeric, generateLetters } from 'utils/string';

import ResponsiveTable from './ResponsiveTable';

export default {
  component: ResponsiveTable,
  parameters: { layout: 'padded' },
  title: 'ResponsiveTable',
};

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
  columns.forEach(column => {
    row[column.dataIndex] = generateAlphaNumeric();
  });
  return row;
});

export const Default = (): React.ReactNode => (
  <ResponsiveTable columns={columns} dataSource={data} />
);
