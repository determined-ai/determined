import { render, screen } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';
import React from 'react';

import EditableMetadata, { Props } from './EditableMetadata';

const DEFAULT_METADATA = { hello: 'world', testing: 'metadata' };

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

const setup = ({
  metadata = DEFAULT_METADATA,
  editing = false,
}: Partial<Props> = {}) => {
  const handleOnChange = jest.fn();
  const view = render(
    <EditableMetadata
      editing={editing}
      metadata={metadata}
      updateMetadata={handleOnChange}
    />,
  );
  return { handleOnChange, view };
};

describe('EditableMetadata', () => {
  it('displays list of metadata', () => {
    setup();

    Object.entries(DEFAULT_METADATA).forEach(([ key, value ]) => {
      expect(screen.getByText(key)).toBeInTheDocument();
      expect(screen.getByText(value)).toBeInTheDocument();
    });
  });

  it('handles metadata addition', async () => {
    const [ additionKey, additionValue ] = [ 'animal', 'fox' ];
    const resultMetadata = {
      ...DEFAULT_METADATA,
      ...Object.fromEntries([ [ additionKey, additionValue ] ]),
    };
    const { handleOnChange } = setup({ editing: true });

    const addRow = screen.getByText('+ Add Row');
    await user.click(addRow);

    const keyInputs = screen.getAllByPlaceholderText('Enter metadata label');
    const keyInput = keyInputs.last();
    await user.click(keyInput);
    await user.type(keyInput, additionKey);

    const valueInput = keyInput.nextSibling as HTMLElement;
    await user.click(valueInput);
    await user.type(valueInput, additionValue);

    expect(handleOnChange).toHaveBeenLastCalledWith(resultMetadata);
  });

  it('handles metadata removal', async () => {
    const metadataArray = Object.entries(DEFAULT_METADATA);
    const removalIndex = Math.floor(Math.random() * metadataArray.length);
    const removalMetadata = metadataArray[removalIndex];
    const resultMetadata = Object.fromEntries(metadataArray.filter(
      ([ key, value ]) => key !== removalMetadata[0] && value !== removalMetadata[1],
    ));
    const { handleOnChange } = setup({ editing: true });

    // There is the "+ Add Row" button in addition to the overflow buttons.
    const buttons = await screen.findAllByRole('button');
    expect(buttons).toHaveLength(metadataArray.length + 1);

    await user.click(buttons[removalIndex]);
    await user.click(await screen.findByText('Delete Row'));
    expect(handleOnChange).toHaveBeenCalledWith(resultMetadata);
  });
});
