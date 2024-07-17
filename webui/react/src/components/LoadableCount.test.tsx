import { render, screen } from '@testing-library/react';
import { Loaded, NotLoaded } from 'hew/utils/loadable';

import LoadableCount from './LoadableCount';

const LABEL_SINGULAR = 'LABEL_SINGULAR';
const LABEL_PLURAL = 'LABEL_PLURAL';

const mocks = vi.hoisted(() => {
  return {
    useMobile: vi.fn().mockReturnValue(false),
  };
});

vi.mock('hooks/useMobile', async (importOriginal) => {
  const useMobile = mocks.useMobile;
  return {
    ...(await importOriginal<typeof import('hooks/useMobile')>()),
    default: useMobile,
  };
});

const setup = (totalCount: number, selectedCount?: number, loaded?: boolean) => {
  const total = loaded ? Loaded(totalCount) : NotLoaded;
  render(
    <LoadableCount
      labelPlural={LABEL_PLURAL}
      labelSingular={LABEL_SINGULAR}
      selectedCount={selectedCount ?? 0}
      total={total}
    />,
  );
};

describe('LoadableCount', () => {
  it('is hidden at mobile resolution', () => {
    mocks.useMobile.mockImplementation(() => true);
    setup(0);
    expect(screen.queryByText(`Loading ${LABEL_PLURAL.toLowerCase()}...`)).not.toBeInTheDocument();
    mocks.useMobile.mockRestore();
  });

  it('shows loading state', () => {
    setup(0);
    expect(screen.getByText(`Loading ${LABEL_PLURAL.toLowerCase()}...`)).toBeInTheDocument();
  });

  it('shows singular count', () => {
    const totalCount = 1;
    setup(totalCount, 0, true);
    expect(screen.getByText(`${totalCount.toLocaleString()} ${LABEL_SINGULAR.toLowerCase()}`)).toBeInTheDocument();
  });

  it('shows plural count', () => {
    const totalCount = 2;
    setup(totalCount, 0, true);
    expect(screen.getByText(`${totalCount.toLocaleString()} ${LABEL_PLURAL.toLowerCase()}`)).toBeInTheDocument();
  });

  it('shows selected count', () => {
    const totalCount = 2;
    const selectedCount = 1;
    setup(totalCount, selectedCount, true);
    expect(screen.getByText(`${selectedCount.toLocaleString()} of ${totalCount.toLocaleString()} ${LABEL_PLURAL.toLowerCase()} selected`)).toBeInTheDocument();
  });
});
