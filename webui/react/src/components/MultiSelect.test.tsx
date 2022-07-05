import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';
import { Select } from 'antd';
import React from 'react';

import { generateAlphaNumeric } from 'shared/utils/string';

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
  const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });
  return { handleOpen, user, view };
};

describe('MultiSelect', () => {
  it('displays label and placeholder', async () => {
    setup();

    await waitFor(() => {
      expect(screen.getByText(LABEL)).toBeInTheDocument();
      expect(screen.getByText(PLACEHOLDER)).toBeInTheDocument();
    });

  });

  it('opens select list', async () => {
    const { handleOpen, user } = setup();
    expect(handleOpen).not.toHaveBeenCalled();
    await waitFor(async () => {
      await user.click(screen.getByText(PLACEHOLDER));
      expect(handleOpen).toHaveBeenCalled();

      expect(screen.getAllByTitle(OPTION_TITLE)).toHaveLength(NUM_OPTIONS);
    });

  });

  it('selects option', async () => {
    const { handleOpen, user } = setup();

    await user.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);

    await user.click(list[0]);

    await waitFor(() => {
      expect(list[0].querySelector('.anticon-check')).toBeInTheDocument();
    });

  });

  it('selects multiple option', async () => {
    const { handleOpen, user } = setup();

    await user.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);

    await user.click(list[0]);
    await user.click(list[1]);

    await waitFor(() => {
      expect(document.querySelectorAll('.anticon-check')).toHaveLength(2);
    });

  });
  it('selects all', async () => {
    const { handleOpen, user } = setup();

    await user.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const all = screen.getByTitle('All Agents');

    await user.click(all);
    await waitFor(() => {
      expect(all.querySelector('.anticon-check')).toBeInTheDocument();
    });

  });

});
