import {
  render,
  screen,
  waitForElementToBeRemoved,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { not } from 'fp-ts/lib/Predicate';
import React from 'react';

import Page from './Page';
import Spinner from './Spinner';

const setup = (
  { disabled, onSaveReturnsError, value } = {
    disabled: false,
    onSaveReturnsError: false,
    value: 'before',
  },
) => {
  const onSave = onSaveReturnsError
    ? jest.fn(() => Promise.resolve(new Error()))
    : jest.fn(() => Promise.resolve());
  const onCancel = jest.fn();
  const { container } = render(
    <Spinner spinning={true}>
      <div>content</div>
    </Spinner>,
  );

  const waitForSpinnerToDisappear = async () =>
    await waitForElementToBeRemoved(
      () => container.getElementsByClassName('ant-spin-spinning')[0],
    );
  return { onCancel, onSave, waitForSpinnerToDisappear };
};

describe('Spinner', () => {
  it('spins', () => {

    setup();
    expect(screen.queryByDisplayValue('content')).toBeNull();

  });

});
