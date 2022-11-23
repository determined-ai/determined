import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import InlineEditor from './InlineEditor';

const setup = (
  { disabled, onSaveReturnsError, value, pattern } = {
    disabled: false,
    onSaveReturnsError: false,
    pattern: new RegExp(''),
    value: 'before',
  },
) => {
  const user = userEvent.setup();
  const onSave = onSaveReturnsError
    ? jest.fn(() => Promise.resolve(new Error()))
    : jest.fn(() => Promise.resolve());
  const onCancel = jest.fn();
  render(
    <InlineEditor
      disabled={disabled}
      pattern={pattern}
      value={value}
      onCancel={onCancel}
      onSave={onSave}
    />,
  );

  return { onCancel, onSave, user };
};

describe('InlineEditor', () => {
  it('displays the value passed as prop', () => {
    setup();
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('should preserves input when focus leaves', async () => {
    const { user } = setup();
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.click(document.body);
    expect(screen.getByRole('textbox')).not.toHaveFocus();
    expect(screen.getByDisplayValue('after')).toBeInTheDocument();
  });

  it('should calls save with input on blur', async () => {
    const { onSave, user } = setup();
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.click(document.body);
    expect(screen.getByRole('textbox')).not.toHaveFocus();
    expect(onSave).toHaveBeenCalledWith('after');
  });

  it('should restores value when save fails', async () => {
    const { onSave, user } = setup({
      disabled: false,
      onSaveReturnsError: true,
      pattern: new RegExp(''),
      value: 'before',
    });
    await user.click(screen.getByRole('textbox'));
    await user.clear(screen.getByRole('textbox'));
    await user.type(screen.getByRole('textbox'), 'after');
    await user.keyboard('{enter}');
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
    const { user } = setup({
      disabled: true,
      onSaveReturnsError: true,
      pattern: new RegExp(''),
      value: 'before',
    });
    await user.type(screen.getByRole('textbox'), 'after');
    expect(screen.getByDisplayValue('before')).toBeInTheDocument();
  });

  it('should ignore keydown event until the IME is confirmed', async () => {
    const { user } = setup();
    const textbox = screen.getByRole('textbox');
    await user.click(textbox);
    await user.clear(textbox);
    await user.type(textbox, 'こんにちは');
    fireEvent.keyDown(textbox, { code: 'Enter', key: 'Enter', keyCode: 229 });
    expect(textbox).toHaveFocus();
    await user.keyboard('{enter}');
    expect(textbox).not.toHaveFocus();
    expect(screen.getByDisplayValue('こんにちは')).toBeInTheDocument();
  });

  it('should RegExp validate input value', async () => {
    const { user } = setup({
      disabled: false,
      onSaveReturnsError: false,
      pattern: new RegExp('^[a-z][a-z0-9\\s]*$', 'i'),
      value: 'Determined',
    });
    const textbox = screen.getByRole('textbox');
    await user.click(textbox);
    await user.type(textbox, '!!');
    await user.keyboard('{enter}');
    expect(screen.queryByDisplayValue('Determined!!')).not.toBeInTheDocument();
    expect(screen.getByDisplayValue('Determined')).toBeInTheDocument();
    await user.click(textbox);
    await user.type(textbox, ' 123');
    await user.keyboard('{enter}');
    expect(screen.getByDisplayValue('Determined 123')).toBeInTheDocument();
  });
});
