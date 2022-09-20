import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';
import React from 'react';

import { generateAlphaNumeric, generateUUID } from 'shared/utils/string';
import { LogLevelFromApi } from 'types';

import LogViewerFilters, { ARIA_LABEL_RESET, Filters, LABELS } from './LogViewerFilters';

const DEFAULT_FILTER_OPTIONS = {
  agentIds: new Array(3).fill('').map(() => `i-${generateAlphaNumeric(17)}`),
  allocationIds: new Array(2).fill('').map((_, i) => `${generateUUID()}.${i}`),
  containerIds: ['', ...new Array(2).fill('').map(() => generateUUID())],
  rankIds: [0, 1, 2],
  sources: ['agent', 'master'],
  stdtypes: ['stdout', 'stderr'],
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
  const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });
  return { handleOnChange, handleOnReset, user, view };
};

describe('LogViewerFilter', () => {
  it('should render all select filters with options', async () => {
    setup(DEFAULT_FILTER_OPTIONS, {});

    await waitFor(() => {
      Object.values(LABELS).forEach((label) => {
        if (!DEFAULT_FILTER_OPTIONS[label]) return;
        const regex = new RegExp(`All ${label}`, 'i');
        expect(screen.queryByText(regex)).toBeInTheDocument();
      });
    });
  });

  it('should render select filters with selected options', async () => {
    const values: Filters = {
      agentIds: [DEFAULT_FILTER_OPTIONS.agentIds[1]],
      allocationIds: [DEFAULT_FILTER_OPTIONS.allocationIds[1]],
      containerIds: [DEFAULT_FILTER_OPTIONS.containerIds[1]],
      levels: [LogLevelFromApi.Info],
      rankIds: [DEFAULT_FILTER_OPTIONS.rankIds[1]],
    };
    setup(DEFAULT_FILTER_OPTIONS, values);

    await waitFor(() => {
      Object.keys(LABELS).forEach((labelKey) => {
        const key = labelKey as keyof Filters;
        if (!values[key]?.length) return;

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
      rankIds: [0, undefined],
    };
    const { user } = setup(values, { ...values, rankIds: [] });

    const agentOption1 = screen.getByText('All Ranks');
    await user.click(agentOption1);
    await waitFor(async () => {
      expect(await screen.findAllByText('0')).toHaveLength(2);
      expect(screen.queryByText('No Rank')).toBeInTheDocument();
    });
  });

  it('should call onChange when options are selected', async () => {
    const { handleOnChange, user } = setup(DEFAULT_FILTER_OPTIONS, {});

    const agentRegex = new RegExp(`All ${LABELS.agentIds}`, 'i');
    const agentOptionText1 = DEFAULT_FILTER_OPTIONS.agentIds[1];
    const agentOptionText2 = DEFAULT_FILTER_OPTIONS.agentIds[2];

    const agent = screen.getByText(agentRegex);
    await user.click(agent);

    const agentOption1 = screen.getByText(agentOptionText1);
    const agentOption2 = screen.getByText(agentOptionText2);
    await user.click(agentOption1);
    await user.click(agentOption2);

    await waitFor(() => {
      /**
       * Since value is not getting updated with the selected options,
       * the results returned by `onChange` do not compound.
       */
      expect(handleOnChange).toHaveBeenCalledWith({ agentIds: [agentOptionText1] });
      expect(handleOnChange).toHaveBeenCalledWith({ agentIds: [agentOptionText2] });
    });
  });

  it('should not show reset button when no filters are set', () => {
    setup(DEFAULT_FILTER_OPTIONS, {});
    expect(screen.queryByText(ARIA_LABEL_RESET)).not.toBeInTheDocument();
  });

  it('should show reset button when filters are set', () => {
    const values = {
      agentIds: [DEFAULT_FILTER_OPTIONS.agentIds[1]],
      containerIds: [DEFAULT_FILTER_OPTIONS.containerIds[1]],
    };
    setup(DEFAULT_FILTER_OPTIONS, values);
    expect(screen.queryByText(ARIA_LABEL_RESET)).toBeInTheDocument();
  });

  it('should call onReset when reset button is clicked', async () => {
    const values = { agentIds: [DEFAULT_FILTER_OPTIONS.agentIds[1]] };
    const { handleOnReset, user } = setup(DEFAULT_FILTER_OPTIONS, values);

    await user.click(screen.getByText(ARIA_LABEL_RESET));

    expect(handleOnReset).toHaveBeenCalled();
  });
});
