// @ts-nocheck
import { fireEvent, render, screen, waitForElementToBeRemoved } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Spinner from './Spinner';

const waitForSpinnerToDisappear = async () =>
  await waitForElementToBeRemoved(() => container.getElementsByClassName('ant-spin-spinning')[0]);

describe('Spinner', () => {
  it('hides while spinning', () => {
    const handleButtonClick = jest.fn(() => console.log('click'));
    render(
      <Spinner spinning={true}>
        <button onClick={handleButtonClick}>click</button>
      </Spinner>
    );
    const button = screen.getByRole('button');
    let error = null;
    try {
      userEvent.click(button);
    } catch (e) {
      error = e;
    }
    expect(error).not.toBeNull();
    expect(handleButtonClick).toHaveBeenCalledTimes(0);
  });

  it('shows when done spinning', () => {
    const handleButtonClick = jest.fn(() => console.log('click'));
    render(
      <Spinner spinning={false}>
        <button onClick={handleButtonClick}>click</button>
      </Spinner>
    );
    const button = screen.getByRole('button');
    userEvent.click(button);
    expect(handleButtonClick).toHaveBeenCalledTimes(1);

  });

  it('displays tip text', () => {
    const handleButtonClick = jest.fn(() => console.log('click'));
    render(
      <Spinner spinning={false}>
        <button onClick={handleButtonClick}>click</button>
      </Spinner>
    );
    // screen.debug();
    // expect(screen.queryByText('content')).toBeInTheDocument();
  });
});
