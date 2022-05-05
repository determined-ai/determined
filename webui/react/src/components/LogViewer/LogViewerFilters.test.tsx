import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import { LogLevelFromApi } from 'types';
import { generateAlphaNumeric, generateUUID } from 'utils/string';

import LogViewerFilters, { ARIA_LABEL_RESET, Filters, LABELS } from './LogViewerFilters';

const DEFAULT_FILTER_OPTIONS = {
  agentIds: new Array(3).fill('').map(() => `i-${generateAlphaNumeric(17)}`),
  allocationIds: new Array(2).fill('').map((_, i) => `${generateUUID()}.${i}`),
  containerIds: [ '', ...new Array(2).fill('').map(() => generateUUID()) ],
  rankIds: [ 0, 1, 2 ],
  sources: [ 'agent', 'master' ],
  stdtypes: [ 'stdout', 'stderr' ],
};

const setup = (filterOptions: Filters, filterValues: Filters) => {
  const handleOnChange = jest.fn();
  const handleOnReset = jest.fn();
  const view = render(
    <LogViewerFilters
      options={filterOptions}
      values={filterValues}
      onChange={handleOnChange}
      onReset={handleOnReset}
    />,
  );
  return { handleOnChange, handleOnReset, view };
};

describe('LogViewerFilter', () => {
  it('should render all filters with options', async () => {
    setup(DEFAULT_FILTER_OPTIONS, {});

    await waitFor(() => {
      Object.values(LABELS).forEach(label => {
        const regex = new RegExp(`All ${label}`, 'i');
        expect(screen.queryByText(regex)).toBeInTheDocument();
      });
    });
  });

  it('should render filters with selected options', async () => {
    const values: Filters = {
      agentIds: [ DEFAULT_FILTER_OPTIONS.agentIds[1] ],
      allocationIds: [ DEFAULT_FILTER_OPTIONS.allocationIds[1] ],
      containerIds: [ DEFAULT_FILTER_OPTIONS.containerIds[1] ],
      levels: [ LogLevelFromApi.Info ],
      rankIds: [ DEFAULT_FILTER_OPTIONS.rankIds[1] ],
    };
    setup(DEFAULT_FILTER_OPTIONS, values);

    await waitFor(() => {
      Object.keys(LABELS).forEach(labelKey => {
        const key = labelKey as keyof Filters;
        if (values[key]?.length ?? 0 === 0) return;

        const regex = new RegExp(`${values[key]?.length} ${LABELS[key]}`, 'i');
        expect(screen.queryByText(regex)).toBeInTheDocument();
      });
    });
  });

  it('should render filters with rank 0 and no rank', async () => {
    const values: Filters = {
      agentIds: [],
      allocationIds: [],
      containerIds: [],
      levels: [],
      rankIds: [ 0, undefined ],
    };
    setup(values, { ...values, rankIds: [] });

    const agentOption1 = screen.getByText('All Ranks');
    userEvent.click(agentOption1, undefined, { skipPointerEventsCheck: true });
    await waitFor(async () => {
      expect(await screen.findAllByText('0')).toHaveLength(2);
      expect(screen.queryByText('No Rank')).toBeInTheDocument();
    });
  });

  it('should call onChange when options are selected', async () => {
    const { handleOnChange } = setup(DEFAULT_FILTER_OPTIONS, {});

    const agentRegex = new RegExp(`All ${LABELS.agentIds}`, 'i');
    const agentOptionText1 = DEFAULT_FILTER_OPTIONS.agentIds[1];
    const agentOptionText2 = DEFAULT_FILTER_OPTIONS.agentIds[2];

    const agent = screen.getByText(agentRegex);
    userEvent.click(agent);

    const agentOption1 = screen.getByText(agentOptionText1);
    const agentOption2 = screen.getByText(agentOptionText2);
    userEvent.click(agentOption1, undefined, { skipPointerEventsCheck: true });
    userEvent.click(agentOption2, undefined, { skipPointerEventsCheck: true });

    await waitFor(() => {
      /**
       * Since value is not getting updated with the selected options,
       * the results returned by `onChange` do not compound.
       */
      expect(handleOnChange).toHaveBeenCalledWith({ agentIds: [ agentOptionText1 ] });
      expect(handleOnChange).toHaveBeenCalledWith({ agentIds: [ agentOptionText2 ] });
    });
  });

  it('should not show reset button when no filters are set', () => {
    setup(DEFAULT_FILTER_OPTIONS, {});
    expect(screen.queryByText(ARIA_LABEL_RESET)).not.toBeInTheDocument();
  });

  it('should show reset button when filters are set', () => {
    const values = {
      agentIds: [ DEFAULT_FILTER_OPTIONS.agentIds[1] ],
      containerIds: [ DEFAULT_FILTER_OPTIONS.containerIds[1] ],
    };
    setup(DEFAULT_FILTER_OPTIONS, values);
    expect(screen.queryByText(ARIA_LABEL_RESET)).toBeInTheDocument();
  });

  it('should call onReset when reset button is clicked', () => {
    const values = { agentIds: [ DEFAULT_FILTER_OPTIONS.agentIds[1] ] };
    const { handleOnReset } = setup(DEFAULT_FILTER_OPTIONS, values);

    userEvent.click(screen.getByText(ARIA_LABEL_RESET));

    expect(handleOnReset).toHaveBeenCalled();
  });
});
