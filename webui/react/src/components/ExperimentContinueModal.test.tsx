import { act, render } from '@testing-library/react';
import { useModal } from 'hew/Modal';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useLayoutEffect } from 'react';

import { generateTestExperimentData } from 'utils/tests/generateTestData';

import ExperimentContinueModalComponent, {
  ContinueExperimentType,
  Props,
} from './ExperimentContinueModal';
import { ThemeProvider } from './ThemeProvider';

const mockUseFeature = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => {
  return {
    default: () => ({
      isOn: mockUseFeature,
    }),
  };
});

const { trial, experiment } = generateTestExperimentData();
const setupTest = (props: Partial<Props> = {}) => {
  const outerRef: { current: null | (() => void) } = { current: null };
  const Wrapper = () => {
    const { Component, open } = useModal(ExperimentContinueModalComponent);

    useLayoutEffect(() => {
      outerRef.current = open;
    });

    return (
      <ThemeProvider>
        <UIProvider theme={DefaultTheme.Light} themeIsDark>
          <Component
            experiment={experiment}
            trial={trial}
            type={ContinueExperimentType.Continue}
            {...props}
          />
        </UIProvider>
      </ThemeProvider>
    );
  };

  const container = render(<Wrapper />);

  return { container, openRef: outerRef };
};

describe('ExperimentContinueModal', () => {
  afterEach(() => {
    mockUseFeature.mockReset();
  });
  it('should render', () => {
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });

    expect(container.queryByText('Continue Trial in New Experiment')).toBeInTheDocument();
  });

  it('should show proper copy when f_flat_runs is on', () => {
    mockUseFeature.mockReturnValue(true);
    const { container, openRef } = setupTest();

    act(() => {
      openRef.current?.();
    });

    expect(container.queryByText('Continue as New Run')).toBeInTheDocument();
  });
});
