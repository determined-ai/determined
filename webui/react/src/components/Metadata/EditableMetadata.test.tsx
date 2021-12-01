import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import { Metadata } from 'types';

import EditableMetadata from './EditableMetadata';

const initMetadata = { hello: 'world', testing: 'metadata' };

const setup = (metadata: Metadata = {}, editing = false) => {
  const handleOnChange = jest.fn();
  const view = render(<EditableMetadata
    editing={editing}
    metadata={metadata}
    updateMetadata={handleOnChange} />);
  return { handleOnChange, view };
};

describe('TagList', () => {
  it('displays list of metadata', () => {
    setup(initMetadata);

    Object.entries(initMetadata).forEach(([ key, value ]) => {
      expect(screen.getByText(key)).toBeInTheDocument();
      expect(screen.getByText(value)).toBeInTheDocument();
    });
  });

  it('handles metadata addition', () => {
    const [ additionKey, additionValue ] = [ 'animal', 'fox' ];
    const resultMetadata = {
      ...initMetadata,
      ...Object.fromEntries([ [ additionKey, additionValue ] ]),
    };
    const { handleOnChange } = setup(initMetadata, true);

    const addRow = screen.getByText('+ Add Row');
    userEvent.click(addRow);

    const keyInputs = screen.getAllByPlaceholderText('Enter metadata label');
    const keyInput = keyInputs.last();
    userEvent.click(keyInput);
    userEvent.type(keyInput, additionKey);

    const valueInput = keyInput.nextSibling as HTMLElement;
    userEvent.click(valueInput);
    userEvent.type(valueInput, additionValue);

    expect(handleOnChange).toHaveBeenLastCalledWith(resultMetadata);
  });

  it('handles metadata removal', async () => {
    const metadataArray = Object.entries(initMetadata);
    const removalIndex = Math.floor(Math.random() * metadataArray.length);
    const removalMetadata = metadataArray[removalIndex];
    const resultMetadata = Object.fromEntries(metadataArray.filter(
      ([ key, value ]) => key !== removalMetadata[0] && value !== removalMetadata[1],
    ));
    const { handleOnChange } = setup(initMetadata, true);

    const metadataRow = screen.getByDisplayValue(removalMetadata[0]).closest('span') as HTMLElement;
    expect(metadataRow).not.toBeNull();

    const openOverflow = within(metadataRow).getByRole('button');
    userEvent.click(openOverflow);

    const deleteRow = screen.getByText('Delete Row');
    await waitFor(() => userEvent.click(deleteRow, undefined, { skipPointerEventsCheck: true }));

    expect(handleOnChange).toHaveBeenCalledWith(resultMetadata);
  });
});
