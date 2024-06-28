import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { Ref } from 'react';

import {
  CloseReason,
  RunMoveWarningFlowRef,
  RunMoveWarningModalComponent,
} from 'components/RunMoveWarningModalComponent';
import { ThemeProvider } from 'components/ThemeProvider';

const setupTest = () => {
  const ref: Ref<RunMoveWarningFlowRef> = { current: null };
  userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <RunMoveWarningModalComponent ref={ref} />
      </ThemeProvider>
    </UIProvider>,
  );

  return {
    ref,
  };
};

describe('RunMoveWarningModalComponent', () => {
  it('is not shown until opened', () => {
    const { ref } = setupTest();

    expect(screen.queryByText('Move Run Dependency Alert')).toBeNull();
    act(() => {
      ref.current?.open();
    });
    expect(screen.queryByText('Move Run Dependency Alert')).not.toBeNull();
  });

  it('closes modal with cancel when closed with the x button', async () => {
    const { ref } = setupTest();

    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    const closeButton = await screen.findByLabelText('Close');
    await userEvent.click(closeButton);
    const closeReason = await lifecycle;
    expect(closeReason).toBe('cancel');
  });

  it('closes modal with cancel when cancel button is pressed', async () => {
    const { ref } = setupTest();

    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    const cancelButton = await screen.findByText('Cancel');
    await userEvent.click(cancelButton);
    const closeReason = await lifecycle;
    expect(closeReason).toBe('cancel');
  });

  it('closes modal with ok when submit is pressed', async () => {
    const { ref } = setupTest();

    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    const okayButton = await screen.findByText('Move runs and searches');
    await userEvent.click(okayButton);
    const closeReason = await lifecycle;
    expect(closeReason).toBe('ok');
  });

  it('closes modal with manual when manually closed with no arg', async () => {
    const { ref } = setupTest();

    let lifecycle: Promise<CloseReason> | undefined;
    act(() => {
      lifecycle = ref.current?.open();
    });
    act(() => {
      ref.current?.close();
    });
    const closeReason = await lifecycle;
    expect(closeReason).toBe('manual');
  });
});
