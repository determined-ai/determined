import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';

import ActionDropdown from 'components/ActionDropdown/ActionDropdown';
import { ConfirmationProvider } from 'components/kit/useConfirm';
import { ValueOf } from 'types';

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

const ACTION_ONE_TEXT = 'Action One';
const ACTION_TWO_TEXT = 'Action Two';

const TestAction = {
  ActionOne: 'Action One',
  ActionTwo: 'Action Two',
} as const;

type TestAction = ValueOf<typeof TestAction>;

const handleActionOne = vi.fn();
const handleActionTwo = vi.fn();

const DropDownContainer = () => {
  const dropDownOnTrigger = () => {
    return {
      [TestAction.ActionOne]: () => handleActionOne(),
      [TestAction.ActionTwo]: () => handleActionTwo(),
    };
  };

  return (
    <ActionDropdown<TestAction>
      actionOrder={[TestAction.ActionOne, TestAction.ActionTwo]}
      id={'test-id'}
      kind="test"
      onError={() => {
        return;
      }}
      onTrigger={dropDownOnTrigger()}
    />
  );
};

const setup = () => {
  const view = render(
    <ConfirmationProvider>
      <DropDownContainer />
    </ConfirmationProvider>,
  );
  return { view };
};

describe('ActionDropdown', () => {
  setup();

  it('should display trigger button', () => {
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('should display actions', async () => {
    setup();

    user.click(screen.getByRole('button'));

    await waitFor(() => {
      expect(screen.getByText(ACTION_ONE_TEXT)).toBeInTheDocument();
      expect(screen.getByText(ACTION_TWO_TEXT)).toBeInTheDocument();
    });
  });

  it('should call dropdown option one function', async () => {
    setup();
    await user.click(screen.getByRole('button'));
    expect(handleActionOne).not.toHaveBeenCalled();
    await user.click(screen.getByText(ACTION_ONE_TEXT));
    expect(handleActionOne).toHaveBeenCalled();
  });

  it('should call dropdown option two function', async () => {
    setup();
    await user.click(screen.getByRole('button'));
    expect(handleActionTwo).not.toHaveBeenCalled();
    await user.click(screen.getByText(ACTION_TWO_TEXT));
    expect(handleActionTwo).toHaveBeenCalled();
  });
});
