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

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

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

describe('MultiSelect', () => {
  it('should display label and placeholder', async () => {
    setup();

    await waitFor(() => {
      expect(screen.getByText(LABEL)).toBeInTheDocument();
      expect(screen.getByText(PLACEHOLDER)).toBeInTheDocument();
    });

  });

  it('should open select list', async () => {
    const { handleOpen } = setup();
    expect(handleOpen).not.toHaveBeenCalled();
    await waitFor(async () => {
      await user.click(screen.getByText(PLACEHOLDER));
      expect(handleOpen).toHaveBeenCalled();

      expect(screen.getAllByTitle(OPTION_TITLE)).toHaveLength(NUM_OPTIONS);
    });

  });

  it('should select option', async () => {
    const { handleOpen } = setup();

    await user.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);

    await user.click(list[0]);

    await waitFor(() => {
      expect(list[0].querySelector('.anticon-check')).toBeInTheDocument();
    });
  });

  it('should select multiple option', async () => {
    const { handleOpen } = setup();

    await user.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);

    await user.click(list[0]);
    await user.click(list[1]);

    await waitFor(() => {
      expect(document.querySelectorAll('.anticon-check')).toHaveLength(2);
    });
  });

  it('should select all', async () => {
    const { handleOpen } = setup();

    await user.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const all = screen.getByTitle('All Agents');

    await user.click(all);
    await waitFor(() => {
      expect(all.querySelector('.anticon-check')).toBeInTheDocument();
    });
  });
});
