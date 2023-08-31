import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import RoutePagination from 'components/RoutePagination';

vi.mock('components/kit/Tooltip');

const FIRST_ID = 6;
const MIDDLE_ID = 66;
const LAST_ID = 666;
const IDS_ARRAY = [FIRST_ID, MIDDLE_ID, LAST_ID];
const TOOLTIP_LABEL = 'unique label name';
const TITLE_PREV = 'Previous Page';
const TITLE_NEXT = 'Next Page';
const TOOLTIP_PREV = 'Previous';
const TOOLTIP_NEXT = 'Next';
const BUTTON_PREV = 'left';
const BUTTON_NEXT = 'right';

const user = userEvent.setup();

const setup = (initialId: number) => {
  const navigateToId = vi.fn();

  render(
    <RoutePagination
      currentId={initialId}
      ids={IDS_ARRAY}
      tooltipLabel={TOOLTIP_LABEL}
      onSelectId={navigateToId}
    />,
  );

  return navigateToId;
};

describe('RoutePagination', () => {
  it('should display both buttons', () => {
    setup(MIDDLE_ID);

    expect(screen.getByRole('listitem', { name: TITLE_PREV })).toBeInTheDocument();
    expect(screen.getByRole('listitem', { name: TITLE_NEXT })).toBeInTheDocument();
  });

  it('should display tooltips on hover', async () => {
    setup(MIDDLE_ID);

    await user.hover(screen.getByRole('button', { name: BUTTON_PREV }));
    expect(screen.getByText(`${TOOLTIP_PREV} ${TOOLTIP_LABEL}`)).toBeInTheDocument();

    await user.hover(screen.getByRole('button', { name: BUTTON_NEXT }));
    expect(screen.getByText(`${TOOLTIP_NEXT} ${TOOLTIP_LABEL}`)).toBeInTheDocument();
  });

  it('should allow user to click to previous page', async () => {
    const navigateToId = setup(MIDDLE_ID);

    await user.click(screen.getByRole('listitem', { name: TITLE_PREV }));
    expect(navigateToId).toHaveBeenCalledWith(FIRST_ID);
  });

  it('should allow user to click to next page', async () => {
    const navigateToId = setup(MIDDLE_ID);

    await user.click(screen.getByRole('listitem', { name: TITLE_NEXT }));
    expect(navigateToId).toHaveBeenCalledWith(LAST_ID);
  });

  it('should disable prev button on first page', async () => {
    const navigateToId = setup(FIRST_ID);

    await user.click(screen.getByRole('listitem', { name: TITLE_PREV }));
    expect(navigateToId).not.toHaveBeenCalled();

    await user.hover(screen.getByRole('button', { name: BUTTON_PREV }));
    expect(screen.queryByText(`${TOOLTIP_PREV} ${TOOLTIP_LABEL}`)).not.toBeInTheDocument();
  });

  it('should disable next button on last page', async () => {
    const navigateToId = setup(LAST_ID);

    await user.click(screen.getByRole('listitem', { name: TITLE_NEXT }));
    expect(navigateToId).not.toHaveBeenCalled();

    await user.hover(screen.getByRole('button', { name: BUTTON_NEXT }));
    expect(screen.queryByText(`${TOOLTIP_NEXT} ${TOOLTIP_LABEL}`)).not.toBeInTheDocument();
  });
});
