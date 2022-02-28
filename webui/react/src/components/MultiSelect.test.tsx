import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Select } from 'antd';
import React from 'react';

import { generateAlphaNumeric } from 'utils/string';

import MultiSelect from './MultiSelect';

const { Option } = Select;

const LABEL = generateAlphaNumeric();
const PLACEHOLDER = generateAlphaNumeric();
const NUM_OPTIONS = 5;
const OPTION_TITLE = 'option';

const setup = () => {
  const handleOpen = jest.fn();
  const view = render(
    <MultiSelect
      itemName="Agent"
      label={LABEL}
      placeholder={PLACEHOLDER}
      onDropdownVisibleChange={handleOpen}>
      {new Array(NUM_OPTIONS).fill(null).map((v, index) => (
        <Option key={index} title={OPTION_TITLE} value={String.fromCharCode(65 + index)}>
          {'Option ' + String.fromCharCode(65 + index)}
        </Option>
      ))}
    </MultiSelect>,
  );
  return { handleOpen, view };
};

describe('SelectFilter', () => {
  it('displays label and placeholder', async () => {
    setup();

    await waitFor(() => {
      expect(screen.getByText(LABEL)).toBeInTheDocument();
      expect(screen.getByText(PLACEHOLDER)).toBeInTheDocument();
    });

  });

  it('opens select list', async () => {
    const { handleOpen } = setup();
    expect(handleOpen).not.toHaveBeenCalled();
    await waitFor(() => {
      userEvent.click(screen.getByText(PLACEHOLDER));
      expect(handleOpen).toHaveBeenCalled();

      expect(screen.getAllByTitle(OPTION_TITLE)).toHaveLength(NUM_OPTIONS);
    });

  });

  it('selects option', async () => {
    const { handleOpen } = setup();

    userEvent.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);

    userEvent.click(list[0], undefined, { skipPointerEventsCheck: true });

    await waitFor(() => {
      expect(list[0].querySelector('.anticon-check')).toBeInTheDocument();
    });

  });

  it('selects multiple option', async () => {
    const { handleOpen } = setup();

    userEvent.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);

    userEvent.click(list[0], undefined, { skipPointerEventsCheck: true });

    userEvent.click(list[1], undefined, { skipPointerEventsCheck: true });

    await waitFor(() => {
      expect(document.querySelectorAll('.anticon-check')).toHaveLength(2);
    });

  });
  it('selects all', async () => {
    const { handleOpen } = setup();

    userEvent.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const all = screen.getByTitle('All Agents');

    userEvent.click(all, undefined, { skipPointerEventsCheck: true });
    await waitFor(() => {
      expect(all.querySelector('.anticon-check')).toBeInTheDocument();
    });

  });

});
