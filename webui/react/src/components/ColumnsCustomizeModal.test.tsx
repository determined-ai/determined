import { render, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useMemo } from 'react';

import ColumnsCustomizeModalComponent from 'components/ColumnsCustomizeModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { DEFAULT_COLUMNS } from 'pages/ExperimentList.settings';
import { camelCaseToSentence, generateAlphaNumeric, sentenceToCamelCase } from 'utils/string';

const BUTTON_TEXT = 'Columns';
const NUM_GENERATED_COLUMNS = 500;

const camelCaseToListItem = (columnName: string) => {
  switch (columnName) {
    case 'id':
      return 'ID';
    case 'startTime':
      return 'Started';
    case 'searcherType':
      return 'Searcher';
    case 'forkedFrom':
      return 'Forked';
    case 'numTrials':
      return 'Trials';
    default:
      return camelCaseToSentence(columnName);
  }
};

const ColumnsButton: React.FC = () => {
  const columns = useMemo(() => {
    const arr: string[] = [...DEFAULT_COLUMNS];
    for (let i = 0; i < NUM_GENERATED_COLUMNS; i++) {
      arr.push(generateAlphaNumeric());
    }
    return arr;
  }, []);

  const ColumnsCustomizeModal = useModal(ColumnsCustomizeModalComponent);

  return (
    <>
      <Button onClick={ColumnsCustomizeModal.open}>{BUTTON_TEXT}</Button>
      <ColumnsCustomizeModal.Component columns={columns} defaultVisibleColumns={DEFAULT_COLUMNS} />
    </>
  );
};

const setup = async () => {
  const user = userEvent.setup();
  const view = render(<ColumnsButton />);

  await user.click(view.getByText(BUTTON_TEXT));

  return { user, view };
};

describe('Columns Customize Modal', () => {
  it('should renders lists', async () => {
    const { view } = await setup();

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');
    const hidden = lists[0];
    const visible = lists[1];

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    const hiddenList = within(hidden).getAllByRole('listitem');
    const visibleList = within(visible).getAllByRole('listitem');

    expect(Array.isArray(hiddenList)).toBeTruthy();
    expect(Array.isArray(visibleList)).toBeTruthy();

    expect(visibleList.map((item) => item.textContent)).toContain(
      camelCaseToListItem(DEFAULT_COLUMNS[0]),
    );
  });

  it('should searche', async () => {
    const { user, view } = await setup();
    const searchTerm = DEFAULT_COLUMNS[1];

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    await user.type(view.getByRole('textbox'), searchTerm);
    expect(view.getByRole('textbox')).toHaveValue(searchTerm);

    await waitFor(() => {
      const visibleList = within(lists[1]).getAllByRole('listitem');
      expect(visibleList[0].textContent).toEqual(camelCaseToListItem(searchTerm));
      expect(visibleList).toHaveLength(1);
    });
  });

  it('should hide column', async () => {
    const { user, view } = await setup();

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    const initialHiddenHeight = parseInt(lists[0].style.height);
    const initialVisibleHeight = parseInt(lists[1].style.height);

    const transferredColumn = within(lists[1]).getAllByRole('listitem')[0];
    await user.click(transferredColumn);

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toBeGreaterThan(initialHiddenHeight);
    });
    expect(parseInt(lists[1].style.height)).toBeLessThan(initialVisibleHeight);
  });

  it('should show column', async () => {
    const { user, view } = await setup();

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    const initialHiddenHeight = parseInt(lists[0].style.height);
    const initialVisibleHeight = parseInt(lists[1].style.height);

    const transferredColumn = within(lists[0]).getAllByRole('listitem')[0];
    await user.click(transferredColumn);

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toBeLessThan(initialHiddenHeight);
    });
    expect(parseInt(lists[1].style.height)).toBeGreaterThan(initialVisibleHeight);
  });

  it('should reset', async () => {
    const { user, view } = await setup();

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    expect(
      within(lists[1])
        .getAllByRole('listitem')
        .map((item) => sentenceToCamelCase(item.textContent ?? '')),
    ).toEqual(DEFAULT_COLUMNS.map((col: string) => camelCaseToListItem(col).toLocaleLowerCase()));

    const transferredColumn = within(lists[1]).getAllByRole('listitem')[0];
    await user.click(transferredColumn);

    await waitFor(() => {
      expect(
        within(lists[1])
          .getAllByRole('listitem')
          .map((item) => sentenceToCamelCase(item.textContent ?? '')),
      ).not.toEqual(DEFAULT_COLUMNS);
    });

    const resetButton = await view.findByText('Reset');
    expect(resetButton).toBeInTheDocument();
    await user.click(resetButton);

    await waitFor(() => {
      expect(
        within(lists[1])
          .getAllByRole('listitem')
          .map((item) => sentenceToCamelCase(item.textContent ?? '')),
      ).toEqual(DEFAULT_COLUMNS.map((col: string) => camelCaseToListItem(col).toLocaleLowerCase()));
    });

    expect(resetButton).not.toBeInTheDocument();
  });

  it('should add all', async () => {
    const { user, view } = await setup();

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    const lineHeight = parseInt(within(lists[0]).getAllByRole('listitem')[0].style.height);

    await user.click(await view.findByText('Add All'));

    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toEqual(0);
      expect(parseInt(lists[1].style.height)).toEqual(
        (NUM_GENERATED_COLUMNS + DEFAULT_COLUMNS.length) * lineHeight,
      );
    });
  });

  it('should remove all', async () => {
    const { user, view } = await setup();

    // Waiting for modal to render.
    await view.findByText('Customize Columns');

    const lists = view.getAllByRole('list');

    // Waiting for list items to render.
    expect((await view.findAllByRole('listitem')).length).toBeGreaterThanOrEqual(
      DEFAULT_COLUMNS.length,
    );

    const lineHeight = parseInt(within(lists[0]).getAllByRole('listitem')[0].style.height);

    await user.click(await view.findByText('Remove All'));

    ///** The reason for the 2 in the line 270 is that the UI never removes all of the options,
    /* it always returns with the id and name. The line 272 is a reflection of the math done on the line 270.
     */
    await waitFor(() => {
      expect(parseInt(lists[0].style.height)).toEqual(
        (NUM_GENERATED_COLUMNS + (DEFAULT_COLUMNS.length - 2)) * lineHeight,
      );
      expect(parseInt(lists[1].style.height)).toEqual(2 * lineHeight);
    });
  });
});
