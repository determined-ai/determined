import { render, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { DEFAULT_COLUMNS } from 'pages/ExperimentList.settings';
import { camelCaseToSentence, generateAlphaNumeric, sentenceToCamelCase } from 'utils/string';

import useModalCustomizeColumns from './useModalCustomizeColumns';

const BUTTON_TEXT = 'Columns';
const NUM_GENERATED_COLUMNS = 50000;

const camelCaseToListItem = (columnName: string) => {
  return columnName === 'id' ? 'ID' : camelCaseToSentence(columnName);
};

const ColumnsButton: React.FC = () => {
  const columns = useMemo(() => {
    const arr = [ ...DEFAULT_COLUMNS ];
    for (let i = 0; i < NUM_GENERATED_COLUMNS; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  const { modalOpen } = useModalCustomizeColumns({
    columns,
    defaultVisibleColumns: DEFAULT_COLUMNS,
  });

  const openModal = useCallback(() => {
    modalOpen({ initialVisibleColumns: DEFAULT_COLUMNS });
  }, [ modalOpen ]);

  return (
    <Button onClick={openModal}>{BUTTON_TEXT}</Button>
  );
};

const setup = async () => {
  const view = render(
    <ColumnsButton />,
  );
  userEvent.click(await view.findByText(BUTTON_TEXT));
  return { view };
};
describe('useCustomizeColumnsModal', () => {
  it('opens modal', async () => {
    const { view } = await setup();

    //waiting for modal to render
    expect(await view.findByText('Customize Columns')).toBeInTheDocument();
  });

  it('closes modal', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    userEvent.click(view.getByText('Cancel'));
    await waitFor(() => {
      expect(view.queryByText('Customize Columns')).not.toBeInTheDocument();
    });
  });

  it('renders lists', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');
    const hidden = lists[0];
    const visible = lists[1];

    expect(Array.isArray(within(hidden).getAllByRole('listitem'))).toBeTruthy();
    expect(Array.isArray(within(visible).getAllByRole('listitem'))).toBeTruthy();

    expect(within(visible).getAllByRole('listitem').map(item => item.textContent))
      .toContain(camelCaseToListItem(DEFAULT_COLUMNS[0]));
  });

  it('searches', async () => {
    const { view } = await setup();
    const searchTerm = DEFAULT_COLUMNS[1];

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    userEvent.type(view.getByRole('textbox'), searchTerm);
    expect(view.getByRole('textbox')).toHaveValue(searchTerm);

    await waitFor(() => {
      const visibleList = within(lists[1]).getAllByRole('listitem');
      expect(visibleList[0].textContent).toEqual(camelCaseToListItem(searchTerm));
      expect(visibleList).toHaveLength(1);
    });
  });

  it('hides column', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');
    const initialHiddenHeight = parseInt(lists[0].style.height);
    const initialVisibleHeight = parseInt(lists[1].style.height);

    const transferredColumn = within(lists[1]).getAllByRole('listitem')[0];
    userEvent.click(transferredColumn);

    await waitFor(() => {
      expect(initialHiddenHeight).toBeLessThan(parseInt(lists[0].style.height));
      expect(initialVisibleHeight).toBeGreaterThan(parseInt(lists[1].style.height));
    });
  });

  it('shows column', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');
    const initialHiddenHeight = parseInt(lists[0].style.height);
    const initialVisibleHeight = parseInt(lists[1].style.height);

    const transferredColumn = within(lists[0]).getAllByRole('listitem')[0];
    userEvent.click(transferredColumn);

    await waitFor(() => {
      expect(initialHiddenHeight).toBeGreaterThan(parseInt(lists[0].style.height));
      expect(initialVisibleHeight).toBeLessThan(parseInt(lists[1].style.height));
    });
  });

  it('resets', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    expect(within(lists[1]).getAllByRole('listitem')
      .map(item => sentenceToCamelCase(item.textContent ?? '')))
      .toEqual(DEFAULT_COLUMNS);

    const transferredColumn = within(lists[1]).getAllByRole('listitem')[0];
    userEvent.click(transferredColumn);

    await waitFor(() => {
      expect(within(lists[1]).getAllByRole('listitem')
        .map(item => sentenceToCamelCase(item.textContent ?? '')))
        .not.toEqual(DEFAULT_COLUMNS);
    });

    const resetButton = await view.findByText('Reset');
    expect(resetButton).toBeInTheDocument();
    userEvent.click(resetButton);

    await waitFor(() => {
      expect(within(lists[1]).getAllByRole('listitem')
        .map(item => sentenceToCamelCase(item.textContent ?? '')))
        .toEqual(DEFAULT_COLUMNS);
    });

    expect(resetButton).not.toBeInTheDocument();
  });

  it('adds all', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');
    const lineHeight = parseInt(within(lists[0]).getAllByRole('listitem')[0].style.height);

    userEvent.click(await view.findByText('Add All'));

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toEqual(0);
      expect(parseInt(lists[1].style.height))
        .toEqual((NUM_GENERATED_COLUMNS + DEFAULT_COLUMNS.length) * lineHeight);
    });
  });

  it('removes all', async () => {
    const { view } = await setup();

    //waiting for modal to render
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');
    const lineHeight = parseInt(within(lists[0]).getAllByRole('listitem')[0].style.height);

    userEvent.click(await view.findByText('Remove All'));

    await waitFor(() => {
      expect(parseInt(lists[0].style.height))
        .toEqual((NUM_GENERATED_COLUMNS + DEFAULT_COLUMNS.length) * lineHeight);
      expect(parseInt(lists[1].style.height)).toEqual(0);
    });
  });
});
