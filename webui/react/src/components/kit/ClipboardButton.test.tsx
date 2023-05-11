import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';

import ClipboardButton, { TOOLTIP_LABEL_DEFAULT } from './ClipboardButton';

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

const CLIPBOARD_CONTENT = 'Copy this into the clipboard.';

const setup = () => {
  const handleCopy = vi.fn();
  const view = render(<ClipboardButton getContent={() => CLIPBOARD_CONTENT} onCopy={handleCopy} />);
  return { handleCopy, view };
};

describe('ClipboardButton', () => {
  it('displays a clipboard button and copies content to clipboard', async () => {
    const { handleCopy } = setup();
    const roleOptions = { name: TOOLTIP_LABEL_DEFAULT };

    await waitFor(() => {
      expect(screen.queryByRole('button', roleOptions)).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', roleOptions));
    expect(handleCopy).toHaveBeenCalled();
  });
});
