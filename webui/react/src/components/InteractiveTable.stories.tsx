import { Meta, Story } from '@storybook/react';
import React, { useCallback, useMemo, useRef } from 'react';

import useSettings, { BaseType, SettingsConfig } from 'hooks/useSettings';
import { generateAlphaNumeric, generateLetters } from 'shared/utils/string';

import InteractiveTable, { InteractiveTableSettings } from './InteractiveTable';

export default {
  argTypes: {
    numRows: { control: { max: 100, min: 0, step: 5, type: 'range' } },
    size: { control: { options: ['default', 'middle', 'small'], type: 'inline-radio' } },
  },
  component: InteractiveTable,
  parameters: { layout: 'padded' },
  title: 'Determined/Tables/InteractiveTable',
} as Meta<typeof InteractiveTable>;

const DEFAULT_COLUMN_WIDTH = 150;

const columns = new Array(20).fill(null).map(() => {
  const str = generateLetters();
  return {
    dataIndex: str,
    defaultWidth: DEFAULT_COLUMN_WIDTH,
    sorter: true,
    title: str,
  };
});

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: columns.map((column) => column.dataIndex),
      key: 'columns',
      storageKey: 'columns',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      defaultValue: columns.map((column) => column.defaultWidth),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      key: 'row',
      type: { baseType: BaseType.String, isArray: true },
    },
  ],
  storagePath: 'storybook',
};

type InteractiveTableProps = React.ComponentProps<typeof InteractiveTable>;

export const Default: Story<InteractiveTableProps & { numRows: number }> = ({
  numRows,
  ...args
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const { settings, updateSettings } = useSettings<InteractiveTableSettings>(config);

  const handleTableRowSelect = useCallback(
    (rowKeys) => {
      updateSettings({ row: rowKeys });
    },
    [updateSettings],
  );

  const data = useMemo(() => {
    return new Array(numRows).fill(null).map(() => {
      const row: Record<string, string> = {};
      columns.forEach((column) => {
        row[column.dataIndex] = generateAlphaNumeric();
      });
      return row;
    });
  }, [numRows]);

  return (
    <div ref={containerRef}>
      <InteractiveTable
        {...args}
        areRowsSelected={!!settings.row}
        columns={columns}
        containerRef={containerRef}
        dataSource={data}
        rowKey={columns[0].title}
        rowSelection={{
          onChange: handleTableRowSelect,
          preserveSelectedRowKeys: true,
          selectedRowKeys: settings.row ?? [],
        }}
        settings={settings}
        updateSettings={updateSettings}
      />
    </div>
  );
};

Default.args = { numRows: 50, showSorterTooltip: false, size: 'small' };
