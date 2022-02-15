import { render, screen, waitFor, within } from '@testing-library/react';
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
  render(
    <ColumnsButton />,
  );
  userEvent.click(await screen.findByText(BUTTON_TEXT));
};

describe('useCustomizeColumnsModal', () => {
  it('opens modal', async () => {
    await setup();

    // waiting for modal to render
    expect(await screen.findByText('Customize Columns')).toBeInTheDocument();
  });

  it('closes modal', async () => {
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    userEvent.click(screen.getByRole('button', { name: /cancel/i }));

    await waitFor(() => {
      expect(screen.queryByText('Customize Columns')).not.toBeInTheDocument();
    });
  });

  it('renders lists', async () => {
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');
    const hidden = lists[0];
    const visible = lists[1];

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

    const hiddenList = within(hidden).getAllByRole('listitem');
    const visibleList = within(visible).getAllByRole('listitem');

    expect(Array.isArray(hiddenList)).toBeTruthy();
    expect(Array.isArray(visibleList)).toBeTruthy();

    expect(visibleList.map(item => item.textContent))
      .toContain(camelCaseToListItem(DEFAULT_COLUMNS[0]));
  });

  it('searches', async () => {
    await setup();
    const searchTerm = DEFAULT_COLUMNS[1];

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

    userEvent.type(screen.getByRole('textbox'), searchTerm);
    expect(screen.getByRole('textbox')).toHaveValue(searchTerm);

    await waitFor(() => {
      const visibleList = within(lists[1]).getAllByRole('listitem');
      expect(visibleList[0].textContent).toEqual(camelCaseToListItem(searchTerm));
      expect(visibleList).toHaveLength(1);
    });
  });

  it('hides column', async () => {
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

    const initialHiddenHeight = parseInt(lists[0].style.height);
    const initialVisibleHeight = parseInt(lists[1].style.height);

    const transferredColumn = within(lists[1]).getAllByRole('listitem')[0];
    userEvent.click(transferredColumn);

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toBeGreaterThan(initialHiddenHeight);
    });
    expect(parseInt(lists[1].style.height)).toBeLessThan(initialVisibleHeight);
  });

  it('shows column', async () => {
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

    const initialHiddenHeight = parseInt(lists[0].style.height);
    const initialVisibleHeight = parseInt(lists[1].style.height);

    const transferredColumn = within(lists[0]).getAllByRole('listitem')[0];
    userEvent.click(transferredColumn);

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toBeLessThan(initialHiddenHeight);
    });
    expect(parseInt(lists[1].style.height)).toBeGreaterThan(initialVisibleHeight);
  });

  it('resets', async () => {
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

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

    const resetButton = await screen.findByText('Reset');
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
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

    const lineHeight = parseInt(within(lists[0]).getAllByRole('listitem')[0].style.height);

    userEvent.click(await screen.findByText('Add All'));

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toEqual(0);
      expect(parseInt(lists[1].style.height))
        .toEqual((NUM_GENERATED_COLUMNS + DEFAULT_COLUMNS.length) * lineHeight);
    });
  });

  it('removes all', async () => {
    await setup();

    // waiting for modal to render
    await screen.findByText('Customize Columns');

    const lists = screen.getAllByRole('list');

    // waiting for list items to render
    expect((await screen.findAllByRole('listitem')).length)
      .toBeGreaterThanOrEqual(DEFAULT_COLUMNS.length);

    const lineHeight = parseInt(within(lists[0]).getAllByRole('listitem')[0].style.height);

    userEvent.click(await screen.findByText('Remove All'));

    await waitFor(() => {
      expect(parseInt(lists[0].style.height))
        .toEqual((NUM_GENERATED_COLUMNS + DEFAULT_COLUMNS.length) * lineHeight);
      expect(parseInt(lists[1].style.height)).toEqual(0);
    });
  });
});
