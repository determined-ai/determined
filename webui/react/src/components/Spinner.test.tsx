import { readFileSync } from 'fs';

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Spinner from './Spinner';

describe('Spinner', () => {
  /*
  * load ant styles into test environment
  * https://github.com/testing-library/jest-dom/issues/113#issuecomment-496971128
  * https://github.com/testing-library/jest-dom
  * /blob/09f7f041805b2a4bcf5ac5c1e8201ee10a69ab9b/src/__tests__/to-have-style.js#L12-L18
  */
  const antdStyleSheet = readFileSync('node_modules/antd/dist/antd.css').toString();
  const style = document.createElement('style');
  style.innerHTML = antdStyleSheet;
  document.body.appendChild(style);

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
    const handleButtonClick = jest.fn();
    render(
      <Spinner spinning={true} tip="Spinner text content">
        <button onClick={handleButtonClick}>click</button>
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
