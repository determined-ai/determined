import { render, screen } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';
import React from 'react';

import { Metadata } from 'types';

import EditableMetadata from './EditableMetadata';

const initMetadata = { hello: 'world', testing: 'metadata' };

const setup = (metadata: Metadata = {}, editing = false) => {
  const handleOnChange = jest.fn();
  const view = render(
    <EditableMetadata
      editing={editing}
      metadata={metadata}
      updateMetadata={handleOnChange}
    />,
  );
  const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });
  return { handleOnChange, user, view };
};

describe('EditableMetadata', () => {
  it('displays list of metadata', () => {
    setup(initMetadata);

    Object.entries(initMetadata).forEach(([ key, value ]) => {
      expect(screen.getByText(key)).toBeInTheDocument();
      expect(screen.getByText(value)).toBeInTheDocument();
    });
  });

  it('handles metadata addition', async () => {
    const [ additionKey, additionValue ] = [ 'animal', 'fox' ];
    const resultMetadata = {
      ...initMetadata,
      ...Object.fromEntries([ [ additionKey, additionValue ] ]),
    };
    const { handleOnChange, user } = setup(initMetadata, true);

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
    const metadataArray = Object.entries(initMetadata);
    const removalIndex = Math.floor(Math.random() * metadataArray.length);
    const removalMetadata = metadataArray[removalIndex];
    const resultMetadata = Object.fromEntries(metadataArray.filter(
      ([ key, value ]) => key !== removalMetadata[0] && value !== removalMetadata[1],
    ));
    const { handleOnChange, user, view } = setup(initMetadata, true);

    await user.click(view.getAllByRole('button')[removalIndex]);
    await user.click(view.getByText('Delete Row'));

    expect(handleOnChange).toHaveBeenCalledWith(resultMetadata);
  });
});
