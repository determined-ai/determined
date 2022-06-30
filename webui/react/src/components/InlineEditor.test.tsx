import {
  render,
  screen,
  waitForElementToBeRemoved,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import InlineEditor from './InlineEditor';

const setup = (
  { disabled, onSaveReturnsError, value } = {
    disabled: false,
    onSaveReturnsError: false,
    value: 'before',
  },
) => {
  // const onSave=jest.fn(async () => {})
  const onSave = onSaveReturnsError
    ? jest.fn(() => Promise.resolve(new Error()))
    : jest.fn(() => Promise.resolve());
  const onCancel = jest.fn();
  const { container } = render(
    <InlineEditor
      disabled={disabled}
      value={value}
      onCancel={onCancel}
      onSave={onSave}
    />,
  );

  const waitForSpinnerToDisappear = async () =>
    await waitForElementToBeRemoved(
      () => container.getElementsByClassName('ant-spin-spinning')[0],
    );
  return { onCancel, onSave, waitForSpinnerToDisappear };
};

describe('InlineEditor', () => {
  it('displays the value passed as prop', () => {
    setup();
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('preserves input when focus leaves', async () => {
    const { waitForSpinnerToDisappear } = setup();
    userEvent.clear(screen.getByRole('textbox'));
    userEvent.type(screen.getByRole('textbox'), 'after');
    userEvent.click(document.body);
    expect(screen.getByRole('textbox')).not.toHaveFocus();
    await waitForSpinnerToDisappear();
    expect(screen.getByDisplayValue('after')).toBeInTheDocument();
  });

  it('calls save with input on blur', async () => {
    const { onSave, waitForSpinnerToDisappear } = setup();
    userEvent.clear(screen.getByRole('textbox'));
    userEvent.type(screen.getByRole('textbox'), 'after');
    userEvent.click(document.body);
    expect(screen.getByRole('textbox')).not.toHaveFocus();
    await waitForSpinnerToDisappear();
    expect(onSave).toHaveBeenCalledWith('after');
  });

  it('restores value when save fails', async () => {
    const { onSave, waitForSpinnerToDisappear } = setup({
      disabled: false,
      onSaveReturnsError: true,
      value: 'before',
    });
    userEvent.clear(screen.getByRole('textbox'));
    userEvent.type(screen.getByRole('textbox'), 'after');
    userEvent.keyboard('{enter}');
    await waitForSpinnerToDisappear();
    expect(onSave).toHaveBeenCalledWith('after');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('calls cancel and restores previous value when esc is pressed', () => {
    const { onCancel } = setup();
    userEvent.clear(screen.getByRole('textbox'));
    userEvent.type(screen.getByRole('textbox'), 'after');
    userEvent.keyboard('{escape}');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
    expect(onCancel).toHaveBeenCalled();
  });

  it('doesnt allow user input when disabled', () => {
    setup({ disabled: true, onSaveReturnsError: true, value: 'before' });
    userEvent.clear(screen.getByRole('textbox'));
    userEvent.type(screen.getByRole('textbox'), 'after');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });
});
