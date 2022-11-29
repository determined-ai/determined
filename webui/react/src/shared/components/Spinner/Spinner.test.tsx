import { readFileSync } from 'fs';

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import Spinner from './Spinner';

jest.useRealTimers(); // This should solve the flakyness around timming out

const spinnerTextContent = 'Spinner Text Content';

const user = userEvent.setup();

const setup = (spinning: boolean) => {
  const handleButtonClick = jest.fn();
  const { container, rerender } = render(
    <Spinner spinning={spinning} tip={spinnerTextContent}>
      <button onClick={handleButtonClick}>click</button>
    </Spinner>,
  );
  return { container, handleButtonClick, rerender };
};

describe('Spinner', () => {
  beforeAll(() => {
    // load Antd StyleSheet
    // Same code is defined in setupTests.ts
    const antdStyleSheet = readFileSync('node_modules/antd/dist/antd.css').toString();
    const style = document.createElement('style');
    style.innerHTML = antdStyleSheet;
    document.body.appendChild(style);
  });

  it('blocks inner content while spinning', async () => {
    const { handleButtonClick } = setup(true);
    const button = screen.getByRole('button');
    let error = null;
    try {
      await user.click(button);
    } catch (e) {
      error = e;
    }
    expect(error).not.toBeNull();
    expect(handleButtonClick).toHaveBeenCalledTimes(0);
  });

  it('doesnt block inner content when not spinning', async () => {
    const { handleButtonClick } = setup(false);
    const button = screen.getByRole('button');
    await user.click(button);
    expect(handleButtonClick).toHaveBeenCalledTimes(1);
  });

  it('displays tip text when spinning', () => {
    setup(true);
    expect(screen.getByText(spinnerTextContent)).toBeInTheDocument();
  });

  it('doesnt display tip text when not spinning', () => {
    setup(false);
    expect(screen.queryByText(spinnerTextContent)).not.toBeInTheDocument();
  });

  it('goes away when spinning is updated to false', () => {
    const { container, rerender } = setup(true);

    expect(container.getElementsByClassName('ant-spin-spinning')[0]).toBeInTheDocument();
    rerender(
      <Spinner spinning={false} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(container.getElementsByClassName('ant-spin-spinning')?.[0]).toBeFalsy();
  });

  it('appears when spinning is updated to false', () => {
    const { container, rerender } = setup(false);
    expect(container.getElementsByClassName('ant-spin-spinning')?.[0]).toBeFalsy();
    rerender(
      <Spinner spinning={true} tip="Spinner text content">
        <button>click</button>
      </Spinner>,
    );
    expect(container.getElementsByClassName('ant-spin-spinning')[0]).toBeInTheDocument();
  });
});
