import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Spinner from './Spinner';

describe('Spinner', () => {
  loadAntdStyleSheet(); // defined in setupTests.ts

  it('blocks inner content while spinning', () => {
    const handleButtonClick = jest.fn();
    render(
      <Spinner spinning={true}>
        <button onClick={handleButtonClick}>click</button>
      </Spinner>,
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

  it('doesnt block inner content when not spinning', () => {
    const handleButtonClick = jest.fn();
    render(
      <Spinner spinning={false}>
        <button onClick={handleButtonClick}>click</button>
      </Spinner>,
    );
    const button = screen.getByRole('button');
    userEvent.click(button);
    expect(handleButtonClick).toHaveBeenCalledTimes(1);

  });

  it('displays tip text when spinning', () => {
    render(
      <Spinner spinning={true} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(screen.getByText('Spinner text content')).toBeInTheDocument();
  });

  it('doesnt display tip text when not spinning', () => {
    render(
      <Spinner spinning={false} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(screen.queryByText('Spinner text content')).not.toBeInTheDocument();

  });

  it('goes away when spinning is updated to false', () => {
    const { container, rerender } = render(
      <Spinner spinning={true} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(container.getElementsByClassName('ant-spin-spinning')[0]).toBeInTheDocument();
    rerender(
      <Spinner spinning={false} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(container.getElementsByClassName('ant-spin-spinning')?.[0]).toBeFalsy();
  });

  it('appears when spinning is updated to false', () => {
    const { container, rerender } = render(
      <Spinner spinning={false} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(container.getElementsByClassName('ant-spin-spinning')?.[0]).toBeFalsy();
    rerender(
      <Spinner spinning={true} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(container.getElementsByClassName('ant-spin-spinning')[0]).toBeInTheDocument();
  });
});
