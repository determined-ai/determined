import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';

import { generateAlphaNumeric } from 'components/kit/internal/functions';

import Select, { Option } from './Select';

const LABEL = generateAlphaNumeric();
const PLACEHOLDER = generateAlphaNumeric();
const NUM_OPTIONS = 5;
const OPTION_TITLE = 'option';

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

const setup = () => {
  const handleOpen = vi.fn();
  const view = render(
    <Select label={LABEL} placeholder={PLACEHOLDER}>
      {new Array(NUM_OPTIONS).fill(null).map((v, index) => (
        <Option key={index} title={OPTION_TITLE} value={String.fromCharCode(65 + index)}>
          {'Option ' + String.fromCharCode(65 + index)}
        </Option>
      ))}
    </Select>,
  );
  return { handleOpen, user, view };
};

describe('Select', () => {
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
    await user.click(screen.getByText(PLACEHOLDER));

    await waitFor(() => {
      expect(screen.queryAllByTitle(OPTION_TITLE)).toHaveLength(NUM_OPTIONS);
    });
  });

  it('selects option', async () => {
    setup();

    await user.click(screen.getByText(PLACEHOLDER));

    const list = screen.getAllByTitle(OPTION_TITLE);
    const firstOption = list[0].textContent ?? '';

    await user.click(list[0]);

    await waitFor(() => {
      expect(document.querySelector('.ant-select-selection-item')?.textContent).toBe(firstOption);
    });
  });

  it('searches', async () => {
    setup();

    await user.click(screen.getByText(PLACEHOLDER));

    const firstOption = screen.getAllByTitle(OPTION_TITLE)[0].textContent ?? '';

    await user.type(screen.getByRole('combobox'), firstOption);

    await waitFor(() => {
      expect(screen.queryAllByTitle(OPTION_TITLE)).toHaveLength(1);
      expect(screen.queryByTitle(OPTION_TITLE)?.textContent).toBe(firstOption);
    });
  });
});
