import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';

import { Metadata } from 'types';

import EditableMetadata, { ADD_ROW_TEXT } from './EditableMetadata';
import {
  DELETE_ROW_LABEL,
  METADATA_KEY_PLACEHOLDER,
  METADATA_VALUE_PLACEHOLDER,
} from './EditableRow';

const initMetadata = { hello: 'world', testing: 'metadata' };

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

const setup = (metadata: Metadata = {}, editing = false) => {
  const handleOnChange = vi.fn();
  const view = render(
    <EditableMetadata editing={editing} metadata={metadata} updateMetadata={handleOnChange} />,
  );
  return { handleOnChange, view };
};

describe('EditableMetadata', () => {
  it('displays list of metadata', () => {
    setup(initMetadata);

    Object.entries(initMetadata).forEach(([key, value]) => {
      expect(screen.getByText(key)).toBeInTheDocument();
      expect(screen.getByText(value)).toBeInTheDocument();
    });
  });

  it('handles metadata addition', async () => {
    const [additionKey, additionValue] = ['animal', 'fox'];
    const resultMetadata = {
      ...initMetadata,
      ...Object.fromEntries([[additionKey, additionValue]]),
    };
    const { handleOnChange } = setup(initMetadata, true);

    const addRow = screen.getByText(ADD_ROW_TEXT);
    await user.click(addRow);

    const keyInputs = screen.getAllByPlaceholderText(METADATA_KEY_PLACEHOLDER);
    const keyInput = keyInputs.last();
    await user.click(keyInput);
    await user.type(keyInput, additionKey);

    const valueInputs = screen.getAllByPlaceholderText(METADATA_VALUE_PLACEHOLDER);
    const valueInput = valueInputs.last();
    await user.click(valueInput);
    await user.type(valueInput, additionValue);

    expect(handleOnChange).toHaveBeenLastCalledWith(resultMetadata);
  });

  it('handles metadata removal', async () => {
    const metadataArray = Object.entries(initMetadata);
    const removalIndex = Math.floor(Math.random() * metadataArray.length);
    const resultMetadata = Object.fromEntries(
      metadataArray.filter((_metadata, idx) => idx !== removalIndex),
    );
    const { handleOnChange, view } = setup(initMetadata, true);

    const actionButton = view.getAllByRole('button', { name: 'action' })[removalIndex];
    await user.click(actionButton);
    user.click(await view.findByText(DELETE_ROW_LABEL, undefined, { container: actionButton }));

    await waitFor(() => {
      expect(handleOnChange).toHaveBeenCalledWith(resultMetadata);
    });
  });
});
