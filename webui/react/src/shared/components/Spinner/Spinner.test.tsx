import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect, useState } from 'react';

import Spinner from './Spinner';

jest.useRealTimers(); // This should solve the flakyness around timming out

const spinnerTextContent = 'Spinner Text Content';

const user = userEvent.setup();
interface Props {
  handleButtonClick: jest.Mock<unknown, unknown[]>;
  spinning: boolean;
}

const SpinnerComponent = ({ spinning, handleButtonClick }: Props) => {
  const [isSpin, setIsSpin] = useState<boolean>(false);

  useEffect(() => {
    setIsSpin(spinning);
  }, [spinning]);

  const onToggle = () => setIsSpin((v) => !v);

  return (
    <>
      <button data-testid="toogle-button" onClick={onToggle}>
        Toggle Spin
      </button>
      <Spinner spinning={isSpin} tip={spinnerTextContent}>
        <button data-testid="inside-button" onClick={handleButtonClick}>
          click
        </button>
      </Spinner>
    </>
  );
};

const setup = (spinning: boolean) => {
  const handleButtonClick = jest.fn();
  const { container } = render(
    <SpinnerComponent handleButtonClick={handleButtonClick} spinning={spinning} />,
  );
  return { container, handleButtonClick };
};

describe('Spinner', () => {
  it('blocks inner content while spinning', async () => {
    const { handleButtonClick } = setup(true);
    const button = await screen.findByTestId('inside-button');
    let error = null;
    try {
      await waitFor(() => user.click(button));
    } catch (e) {
      error = e;
    }
    expect(error).not.toBeNull();
    expect(handleButtonClick).toHaveBeenCalledTimes(0);
  });

  it('doesnt block inner content when not spinning', async () => {
    const { handleButtonClick } = setup(false);
    const button = screen.getByTestId('inside-button');
    await user.click(button);
    expect(handleButtonClick).toHaveBeenCalledTimes(1);
  });

  it('displays tip text when spinning', async () => {
    setup(true);
    expect(await screen.findByText(spinnerTextContent)).toBeInTheDocument();
  });

  it('doesnt display tip text when not spinning', () => {
    setup(false);
    expect(screen.queryByText(spinnerTextContent)).not.toBeInTheDocument();
  });

  it('goes away when spinning is updated to false', async () => {
    const { container } = setup(true);

    await waitFor(() => {
      expect(container.getElementsByClassName('ant-spin-spinning')[0]).toBeInTheDocument();
    });
    await user.click(screen.getByTestId('toogle-button'));
    await waitFor(() => {
      expect(container.getElementsByClassName('ant-spin-spinning')?.[0] ?? false).toBeFalsy();
    });
  });

  it('appears when spinning is updated to false', async () => {
    const { container } = setup(false);
    expect(container.getElementsByClassName('ant-spin-spinning')?.[0]).toBeFalsy();
    await user.click(screen.getByTestId('toogle-button'));
    await waitFor(() => {
      expect(container.getElementsByClassName('ant-spin-spinning')[0]).toBeInTheDocument();
    });
  });
});
