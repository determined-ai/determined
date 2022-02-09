import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Select } from 'antd';
import React from 'react';

import { generateAlphaNumeric } from 'utils/string';

import SelectFilter from './SelectFilter';

const { Option } = Select;

const LABEL = generateAlphaNumeric();
const PLACEHOLDER = generateAlphaNumeric();
const NUM_OPTIONS = 5;
const OPTION_TITLE = 'option';

const setup = () => {
  const handleOpen = jest.fn();
  const view = render(
    <SelectFilter label={LABEL} open placeholder={PLACEHOLDER} onDropdownVisibleChange={handleOpen}>
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
  it('displays label and placeholder', () => {
    const { view } = setup();

    expect(view.getByText(LABEL)).toBeInTheDocument();
    expect(view.getByText(PLACEHOLDER)).toBeInTheDocument();
  });

  it('opens select list', () => {
    const { handleOpen, view } = setup();

    expect(handleOpen).not.toHaveBeenCalled();
    userEvent.click(view.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    expect(view.getAllByTitle(OPTION_TITLE)).toHaveLength(NUM_OPTIONS);
  });

  it('selects option', () => {
    const { handleOpen, view } = setup();

    userEvent.click(view.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const list = view.getAllByTitle(OPTION_TITLE);
    const firstOption = view.getAllByTitle(OPTION_TITLE)[0].textContent ?? '';

    userEvent.click(list[0], undefined, { skipPointerEventsCheck: true });

    expect(document.querySelector('.ant-select-selection-item')?.textContent).toBe(firstOption);
  });

  it('searches', () => {
    const { handleOpen, view } = setup();

    userEvent.click(view.getByText(PLACEHOLDER));
    expect(handleOpen).toHaveBeenCalled();

    const firstOption = view.getAllByTitle(OPTION_TITLE)[0].textContent ?? '';

    userEvent.type(view.getByRole('combobox'), firstOption);

    expect(view.queryAllByTitle(OPTION_TITLE)).toHaveLength(1);
  });
});
