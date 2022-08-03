import {
  fireEvent,
  render,
  screen,
  waitForElementToBeRemoved,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import InlineEditor from './InlineEditor';

const setup = ({ disabled, onSaveReturnsError, value } = {
  disabled: false,
  onSaveReturnsError: false,
  value: 'before',
}) => {
  const user = userEvent.setup();
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

  const waitForSpinnerToDisappear = async () => {
    if (container.querySelector('.ant-spin-spinning') == null) return;
    await waitForElementToBeRemoved(container.querySelector('.ant-spin-spinning'));
  };

  return { onCancel, onSave, user, waitForSpinnerToDisappear };
};

describe('InlineEditor', () => {
  it('displays the value passed as prop', () => {
    setup();
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('should preserves input when focus leaves', async () => {
    const { waitForSpinnerToDisappear, user } = setup();
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.click(document.body);
    expect(screen.getByRole('textbox')).not.toHaveFocus();
    await waitForSpinnerToDisappear();
    expect(screen.getByDisplayValue('after')).toBeInTheDocument();
  });

  it('should calls save with input on blur', async () => {
    const { onSave, waitForSpinnerToDisappear, user } = setup();
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.click(document.body);
    expect(screen.getByRole('textbox')).not.toHaveFocus();
    await waitForSpinnerToDisappear();
    expect(onSave).toHaveBeenCalledWith('after');
  });

  it('should restores value when save fails', async () => {
    const { onSave, waitForSpinnerToDisappear, user } = setup({
      disabled: false,
      onSaveReturnsError: true,
      value: 'before',
    });
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.keyboard('{enter}');
    await waitForSpinnerToDisappear();
    expect(onSave).toHaveBeenCalledWith('after');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('should calls cancel and restores previous value when esc is pressed', async () => {
    const { onCancel, user } = setup();
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.keyboard('{escape}');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
    expect(onCancel).toHaveBeenCalled();
  });

  it('should not allow user input when disabled', async () => {
    const { user } = setup({ disabled: true, onSaveReturnsError: true, value: 'before' });
    await user.type(screen.getByRole('textbox'), 'after');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('should ignore keydown event until the IME is confirmed', async () => {
    const { waitForSpinnerToDisappear, user } = setup();
    const textbox = screen.getByRole('textbox');
    await user.click(textbox);
    await user.clear(textbox);
    await user.type(textbox, 'こんにちは');
    fireEvent.keyDown(textbox, {
      charCode: 229,
      code: 'Enter',
      key: 'Enter',
      keyCode: 229,
    });
    expect(textbox).toHaveFocus();
    await user.keyboard('{enter}');
    await waitForSpinnerToDisappear();
    expect(textbox).not.toHaveFocus();
    expect(screen.getByDisplayValue('こんにちは')).toBeInTheDocument();
  });
});
