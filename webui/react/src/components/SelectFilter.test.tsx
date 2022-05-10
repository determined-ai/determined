import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Select } from 'antd';
import React from 'react';

import { generateAlphaNumeric } from 'shared/utils/string';

import SelectFilter from './SelectFilter';

const { Option } = Select;

const LABEL = generateAlphaNumeric();
const PLACEHOLDER = generateAlphaNumeric();
const NUM_OPTIONS = 5;
const OPTION_TITLE = 'option';

const setup = () => {
  const handleOpen = jest.fn();
  const view = render(
    <SelectFilter label={LABEL} placeholder={PLACEHOLDER} onDropdownVisibleChange={handleOpen}>
      {new Array(NUM_OPTIONS).fill(null).map((v, index) => (
        <Option key={index} title={OPTION_TITLE} value={String.fromCharCode(65 + index)}>
          {'Option ' + String.fromCharCode(65 + index)}
        </Option>
      ))}
    </SelectFilter>,
  );
  return { handleOpen, view };
};

describe('SelectFilter', () => {
  it('displays label and placeholder', async () => {
    setup();

    await waitFor(() => {
      expect(screen.queryByText(LABEL)).toBeInTheDocument();
      expect(screen.queryByText(PLACEHOLDER)).toBeInTheDocument();
    });
  });

  it('opens select list', async () => {
    const { handleOpen } = setup();

    expect(handleOpen).not.toHaveBeenCalled();
    userEvent.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    await waitFor(() => {
      expect(screen.queryAllByTitle(OPTION_TITLE)).toHaveLength(NUM_OPTIONS);
    });
  });

  it('selects option', async () => {
    const { handleOpen } = setup();

    userEvent.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = screen.getAllByTitle(OPTION_TITLE);
    const firstOption = list[0].textContent ?? '';

    /**
     * With Ant Design v4 the select dropdown box container has an issue of having the style
     * set as "opacity: 0; pointer-events: none;" which prevents the default `userEvent.click()`
     * from working, because it checks for pointer-events.
     * https://github.com/ant-design/ant-design/issues/23009#issuecomment-929766415
     */
    userEvent.click(list[0], undefined, { skipPointerEventsCheck: true });

    await waitFor(() => {
      expect(document.querySelector('.ant-select-selection-item')?.textContent).toBe(firstOption);
    });
  });

  it('searches', async () => {
    const { handleOpen } = setup();

    userEvent.click(screen.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const firstOption = screen.getAllByTitle(OPTION_TITLE)[0].textContent ?? '';

    userEvent.type(screen.getByRole('combobox'), firstOption);

    await waitFor(() => {
      expect(screen.queryAllByTitle(OPTION_TITLE)).toHaveLength(1);
      expect(screen.queryByTitle(OPTION_TITLE)?.textContent).toBe(firstOption);
    });
  });
});
