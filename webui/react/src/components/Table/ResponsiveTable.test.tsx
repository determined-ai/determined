import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ColumnType } from 'antd/es/table';
import { FilterDropdownProps } from 'antd/es/table/interface';
import React from 'react';

import { getFullPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { Pagination, RecordKey, UnknownRecord, ValueOf } from 'types';
import { alphaNumericSorter, numericSorter } from 'utils/sort';
import { generateAlphaNumeric } from 'utils/string';

import ResponsiveTable from './ResponsiveTable';
import TableFilterDropdown, {
  ARIA_LABEL_APPLY,
  ARIA_LABEL_CONTAINER,
  ARIA_LABEL_INPUT,
} from './TableFilterDropdown';

const ColumnValueType = {
  Decimal: 'decimal',
  Integer: 'integer',
  String: 'string',
} as const;

type ColumnValueType = ValueOf<typeof ColumnValueType>;

interface ColumnConfig {
  length?: number;
  max?: number;
  min?: number;
  type: ColumnValueType;
  undefinedRate?: number;
}

interface TableItem {
  age?: number;
  id: string;
  key: string;
  name: string;
}

const ARIA_LABEL_FILTER = 'filter';
const ARIA_LABEL_FILTER_CLEAR = 'close-circle';
const DATA_ENTRY_COUNT = 100;

const columns: ColumnType<TableItem>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: (a: TableItem, b: TableItem) => alphaNumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    dataIndex: 'name',
    key: 'name',
    sorter: (a: TableItem, b: TableItem) => alphaNumericSorter(a.name, b.name),
    title: 'Name',
  },
  {
    dataIndex: 'age',
    key: 'age',
    sorter: (a: TableItem, b: TableItem) => numericSorter(a.age, b.age),
    title: 'Age',
  },
];

const columnConfig: Record<RecordKey, ColumnConfig> = {
  age: {
    max: 100,
    min: 0,
    type: ColumnValueType.Integer,
    undefinedRate: 0.1,
  },
  id: {
    length: 8,
    type: ColumnValueType.String,
  },
  name: {
    length: 16,
    type: ColumnValueType.String,
    undefinedRate: 0.1,
  },
};

const generateTableData = (entries: number): TableItem[] => {
  return new Array(entries).fill(null).map((entry, index) => {
    const row: UnknownRecord = { key: index };

    for (const column of columns) {
      const key = (column.key ?? column.dataIndex) as keyof TableItem;
      if (!key) continue;

      const config = columnConfig[key];
      const max = config.max;
      const min = config.min ?? 0;
      const undefinedRate = config.undefinedRate;

      if (undefinedRate && Math.random() < undefinedRate) continue;

      if (config.type === ColumnValueType.Decimal && max) {
        const decimal = Math.random() * (max - min) + min;
        row[key] = decimal;
      } else if (config.type === ColumnValueType.Integer && max) {
        const integer = Math.floor(Math.random() * (max - min)) + min;
        row[key] = integer;
      } else if (config.type === ColumnValueType.String) {
        row[key] = generateAlphaNumeric(config.length);
      }
    }

    return row as unknown as TableItem;
  });
};

const setup = (options?: { pagination?: Pagination }) => {
  const onChange = vi.fn();
  const onIdFilter = vi.fn();
  const onIdReset = vi.fn();

  const data = generateTableData(DATA_ENTRY_COUNT);
  const idList = data.map((row) => row.id);
  const paginationConfig = options?.pagination
    ? getFullPaginationConfig(options?.pagination, data.length)
    : undefined;

  const idFilterDropdown = (filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={[]}
      onFilter={onIdFilter}
      onReset={onIdReset}
    />
  );

  // Apply id column filter config.
  const newColumns = columns.map((column) => {
    if (column.key === 'id') {
      column.filterDropdown = idFilterDropdown;
      column.filters = idList.map((id) => ({ text: id, value: id }));
    }
    return column;
  });

  const view = render(
    <ResponsiveTable<TableItem>
      columns={newColumns}
      dataSource={data}
      pagination={paginationConfig}
      onChange={onChange}
    />,
  );

  const rerender = () =>
    view.rerender(
      <ResponsiveTable<TableItem>
        columns={columns}
        dataSource={data}
        pagination={paginationConfig}
        onChange={onChange}
      />,
    );

  const user = userEvent.setup();

  return {
    data,
    handlers: { onChange, onIdFilter, onIdReset },
    paginationConfig,
    rerender,
    user,
    view,
  };
};

describe('ResponsiveTable', () => {
  it('loads the next table page', () => {
    const pagination = { limit: MINIMUM_PAGE_SIZE, offset: 0 };
    const { handlers, paginationConfig } = setup({ pagination });

    screen.getByRole('button', { name: 'right' }).click();

    /*
     * `toHaveBeenCalledWith()` requires all the params to be matching,
     * so in the case of the `onChange` callback, it is called with 4 parameters.
     * All 4 parameters are objects and so a `expect.objectContaining({})` is
     * provided to indicate that we just care that it's an object but not expecting
     * the sub properties to match and shape or data type. Without it, the assertion
     * will fail and complain about not matching the shape of the callback parameters.
     */
    expect(handlers.onChange).toHaveBeenCalledWith(
      expect.objectContaining({ current: (paginationConfig?.current ?? 0) + 1 }),
      expect.objectContaining({}),
      expect.objectContaining({}),
      expect.objectContaining({}),
    );
  });

  it('loads the previous table page', () => {
    const pagination = { limit: MINIMUM_PAGE_SIZE, offset: MINIMUM_PAGE_SIZE };
    const { handlers, paginationConfig } = setup({ pagination });

    screen.getByRole('button', { name: 'left' }).click();

    expect(handlers.onChange).toHaveBeenCalledWith(
      expect.objectContaining({ current: (paginationConfig?.current ?? 0) - 1 }),
      expect.objectContaining({}),
      expect.objectContaining({}),
      expect.objectContaining({}),
    );
  });

  it('sorts by table column, both ascending and descending', async () => {
    const { handlers } = setup();

    for (const column of columns) {
      const key = (column.key ?? column.dataIndex) as keyof TableItem;
      if (!key || typeof column.title !== 'string') continue;

      // Click on the column sorter.
      screen.getByText(column.title).click();

      expect(handlers.onChange).toHaveBeenCalledWith(
        expect.objectContaining({}),
        expect.objectContaining({}),
        expect.objectContaining({ columnKey: key, order: 'ascend' }),
        expect.objectContaining({}),
      );

      // Click on the column sorter again to get reverse order.
      await userEvent.click(screen.getByText(column.title));

      expect(handlers.onChange).toHaveBeenCalledWith(
        expect.objectContaining({}),
        expect.objectContaining({}),
        expect.objectContaining({ columnKey: key, order: 'descend' }),
        expect.objectContaining({}),
      );
    }
  });

  it('filter table by ID', async () => {
    const PICK_COUNT = 3;
    const { data, handlers, user } = setup();

    // Click on the ID column filter.
    screen.getByLabelText(ARIA_LABEL_FILTER).click();

    /*
     * This hack required to override animation style properties in antd.
     * Waiting for the animation to complete does not work.
     */
    const dropdown = (await screen.findByLabelText(ARIA_LABEL_CONTAINER)).closest(
      '.ant-dropdown',
    ) as HTMLElement;
    dropdown.style.removeProperty('opacity');
    dropdown.style.removeProperty('pointer-events');
    expect(dropdown).not.toHaveStyle({ opacity: 0, pointerEvents: 'none' });

    // Randomly pick an ID and type it into filter search, to select the ID filter option.
    const idSearchList: string[] = [];
    for (let i = 0; i < PICK_COUNT; i++) {
      // Ensure a unique id is picked to avoid conflicts in `idSearchList`.
      let id = data.random().id;
      while (idSearchList.includes(id)) id = data.random().id;
      idSearchList.push(id);

      const filterInput = screen.getByLabelText(ARIA_LABEL_INPUT);
      await user.click(filterInput);
      await user.type(filterInput, `${id}{enter}`);

      const filterContainer = screen.getByLabelText(ARIA_LABEL_CONTAINER);
      const option = within(filterContainer).getByText(id).closest('[data-value]');
      if (option) await user.click(option);

      const filterInputClear = screen.getByLabelText(ARIA_LABEL_FILTER_CLEAR);
      await user.click(filterInputClear);
    }

    // Apply filter.
    const filterApply = screen.getByLabelText(ARIA_LABEL_APPLY);
    await user.click(filterApply);

    expect(handlers.onIdFilter).toHaveBeenCalledWith(idSearchList);
  });
});
