import { act, render } from '@testing-library/react';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';
import { useLayoutEffect } from 'react';

import { generateTestExperimentData } from 'utils/tests/generateTestData';

import CheckpointModalComponent, { Props } from './CheckpointModal';
import { ThemeProvider } from './ThemeProvider';

const mockUseFeature = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => {
  return {
    default: () => ({
      isOn: mockUseFeature,
    }),
  };
});

const TITLE = 'MODAL TITLE';
const { experiment, checkpoint } = generateTestExperimentData();

const setupTest = (props: Partial<Props> = {}) => {
  const outerRef: { current: null | (() => void) } = { current: null };
  const Wrapper = () => {
    const { Component, open } = useModal(CheckpointModalComponent);

    useLayoutEffect(() => {
      outerRef.current = open;
    });

    return (
      <ThemeProvider>
        <UIProvider theme={DefaultTheme.Light} themeIsDark>
          <ConfirmationProvider>
            <Component
              checkpoint={checkpoint}
              config={experiment.config}
              title={TITLE}
              {...props}
            />
          </ConfirmationProvider>
        </UIProvider>
      </ThemeProvider>
    );
  };

  const container = render(<Wrapper />);

  return { container, openRef: outerRef };
};

describe('CheckpointModal', () => {
  afterEach(() => {
    mockUseFeature.mockReset();
  });
  it('shows the component when open', () => {
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });

    expect(container.queryByText(TITLE)).toBeInTheDocument();
    expect(container.queryByText(TITLE)).toBeVisible();
  });

  it('shows run copy when runs flag is on', () => {
    mockUseFeature.mockReturnValue(true);
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });
    const searchCopy = container.queryByText(`Search ${checkpoint.experimentId}`);
    const runCopy = container.queryByText(`Run ${checkpoint.trialId}`);
    expect(searchCopy).toBeInTheDocument();
    expect(searchCopy).toBeVisible();
    expect(runCopy).toBeInTheDocument();
    expect(runCopy).toBeVisible();
  });
});
